package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Format   string    `json:"format,omitempty"` // e.g., "json"
}

type ollamaChatResponse struct {
	Model     string  `json:"model"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
	Error     string  `json:"error,omitempty"`
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
		return fmt.Errorf("ollama returned unexpected status: %s", resp.Status)
	}

	// For a robust implementation, we would parse the JSON and check if `model` exists
	// in the `models` array. For now, returning nil means Ollama is mostly healthy.
	return nil
}

// Generate makes a non-streaming chat completion request.
func (p *OllamaProvider) Generate(ctx context.Context, req CompletionRequest) (string, error) {
	ollamaReq := ollamaChatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
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

	go func() {
		defer close(tokenChan)
		defer close(errChan)

		ollamaReq := ollamaChatRequest{
			Model:    req.Model,
			Messages: req.Messages,
			Stream:   true,
		}
		if req.JSONMode {
			ollamaReq.Format = "json"
		}

		body, err := json.Marshal(ollamaReq)
		if err != nil {
			errChan <- fmt.Errorf("failed to marshal streaming request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			errChan <- err
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := p.HTTPClient.Do(httpReq)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errChan <- fmt.Errorf("ollama backend error, status: %d", resp.StatusCode)
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chunk ollamaChatResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				errChan <- fmt.Errorf("failed to decode chunk: %w", err)
				return
			}

			if chunk.Error != "" {
				errChan <- fmt.Errorf("ollama stream error: %s", chunk.Error)
				return
			}

			// Send the incremental token text (if any)
			if chunk.Message.Content != "" {
				// Don't block forever if context is canceled
				select {
				case <-ctx.Done():
					return
				case tokenChan <- chunk.Message.Content:
				}
			}

			if chunk.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	return tokenChan, errChan
}
