package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// OllamaProvider implements the Provider interface for the Ollama API.
type OllamaProvider struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewOllamaProvider initializes an Ollama client.
func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

type ollamaChatRequest struct {
	Model    string       `json:"model"`
	Messages []Message    `json:"messages"`
	Stream   bool         `json:"stream"`
	Format   string       `json:"format,omitempty"` // e.g., "json"
	Tools    []Definition `json:"tools,omitempty"`
}

type ollamaChatResponse struct {
	Model   string  `json:"model"`
	Message Message `json:"message"`
	Done    bool    `json:"done"`
	Error   string  `json:"error,omitempty"`
}

// Preflight checks if Ollama is running and the model is available locally.
func (p *OllamaProvider) Preflight(ctx context.Context, model string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.BaseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("could not create preflight request: %w", err)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("is ollama running? failed to connect to %s: %w", p.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return fmt.Errorf("failed to decode tags: %w", err)
	}

	for _, m := range tags.Models {
		if m.Name == model {
			return nil
		}
	}

	return fmt.Errorf("model %q not found in Ollama. Please run 'ollama pull %s' to download it", model, model)
}

func (p *OllamaProvider) handleError(resp *http.Response) error {
	var ollamaErr struct {
		Error string `json:"error"`
	}

	// Try to decode error message from body
	_ = json.NewDecoder(resp.Body).Decode(&ollamaErr)

	msg := ollamaErr.Error
	if msg == "" {
		msg = resp.Status
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("Ollama model not found or URL incorrect (404). Details: %s", msg)
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("Ollama authentication error (403/401). Details: %s", msg)
	case http.StatusInternalServerError:
		return fmt.Errorf("Ollama internal server error (500). Details: %s", msg)
	default:
		return fmt.Errorf("Ollama returned error (status %d): %s", resp.StatusCode, msg)
	}
}

// Generate makes a non-streaming chat completion request.
func (p *OllamaProvider) Generate(ctx context.Context, req CompletionRequest) (string, error) {
	ollamaReq := ollamaChatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
		Tools:    req.Tools,
	}
	if req.JSONMode {
		ollamaReq.Format = "json"
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.HTTPClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", p.handleError(resp)
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if ollamaResp.Error != "" {
		return "", fmt.Errorf("ollama error: %s", ollamaResp.Error)
	}

	return ollamaResp.Message.Content, nil
}

// GenerateStream provides back tokens over a channel as they arrive from Ollama.
func (p *OllamaProvider) GenerateStream(ctx context.Context, req CompletionRequest) (<-chan string, <-chan error) {
	tokenChan := make(chan string)
	errChan := make(chan error, 1)

	slog.Info("Starting Ollama stream", "model", req.Model, "messages", len(req.Messages), "tools", len(req.Tools))

	go func() {
		defer close(tokenChan)
		defer close(errChan)

		ollamaReq := ollamaChatRequest{
			Model:    req.Model,
			Messages: req.Messages,
			Stream:   true,
			Tools:    req.Tools,
		}
		if req.JSONMode {
			ollamaReq.Format = "json"
		}

		body, err := json.Marshal(ollamaReq)
		if err != nil {
			slog.Error("Failed to marshal request", "error", err)
			errChan <- fmt.Errorf("failed to marshal streaming request: %w", err)
			return
		}

		slog.Debug("Request body", "body", string(body))

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			slog.Error("Failed to create HTTP request", "error", err)
			errChan <- err
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := p.HTTPClient.Do(httpReq)
		if err != nil {
			slog.Error("HTTP request failed", "error", err)
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("Ollama returned error status", "status", resp.StatusCode)
			errChan <- p.handleError(resp)
			return
		}

		slog.Debug("Stream started, reading chunks")
		scanner := bufio.NewScanner(resp.Body)
		chunkCount := 0
		var collectedToolCalls []ToolCall
		
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			chunkCount++
			slog.Debug("Received chunk", "count", chunkCount, "line", string(line))

			var chunk ollamaChatResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				slog.Error("Failed to decode chunk", "error", err, "line", string(line))
				errChan <- fmt.Errorf("failed to decode chunk: %w", err)
				return
			}

			if chunk.Error != "" {
				slog.Error("Ollama returned error", "error", chunk.Error)
				errChan <- fmt.Errorf("ollama stream error: %s", chunk.Error)
				return
			}

			// Collect tool calls
			if len(chunk.Message.ToolCalls) > 0 {
				collectedToolCalls = append(collectedToolCalls, chunk.Message.ToolCalls...)
				slog.Info("Tool calls detected", "count", len(chunk.Message.ToolCalls))
			}

			// Send the incremental token text (if any)
			if chunk.Message.Content != "" {
				slog.Debug("Sending token", "content", chunk.Message.Content)
				select {
				case <-ctx.Done():
					slog.Debug("Context cancelled, stopping stream")
					return
				case tokenChan <- chunk.Message.Content:
				}
			}

			if chunk.Done {
				// If we have tool calls but no content, send tool call info as special message
				if len(collectedToolCalls) > 0 && chunk.Message.Content == "" {
					slog.Info("Stream completed with tool calls", "tool_count", len(collectedToolCalls))
					// Send a special marker for tool calls
					toolCallJSON, _ := json.Marshal(collectedToolCalls)
					select {
					case tokenChan <- fmt.Sprintf("__TOOL_CALL__%s", string(toolCallJSON)):
					default:
					}
				}
				slog.Info("Stream completed", "total_chunks", chunkCount)
				return
			}
		}

		if err := scanner.Err(); err != nil {
			slog.Error("Scanner error", "error", err)
			errChan <- err
		}
	}()

	return tokenChan, errChan
}
