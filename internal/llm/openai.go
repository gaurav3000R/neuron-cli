package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type OpenAIProvider struct {
	BaseURL      string
	APIKey       string
	HTTPClient   *http.Client
	providerName string
}

func NewOpenAIProvider(name, baseURL, apiKey string) *OpenAIProvider {
	if name == "" {
		name = "OpenAI"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		BaseURL:      baseURL,
		APIKey:       apiKey,
		HTTPClient:   &http.Client{},
		providerName: name,
	}
}

func (p *OpenAIProvider) Name() string {
	return p.providerName
}

type openAIChatRequest struct {
	Model    string       `json:"model"`
	Messages []Message    `json:"messages"`
	Stream   bool         `json:"stream"`
	Tools    []Definition `json:"tools,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		Delta struct {
			Content   string     `json:"content,omitempty"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta,omitempty"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *OpenAIProvider) Preflight(ctx context.Context, model string) error {
	if p.APIKey == "" {
		return fmt.Errorf("API key is required for OpenAI-compatible provider")
	}
	// We'll skip the `/models` endpoint check here as many OpenAI-compatible endpoints
	// (like Google Gemini and HuggingFace Serverless Inference) either:
	// 1. Don't implement `/models` endpoint
	// 2. Reject it with a 400/404
	// 3. Wait for the actual chat completion endpoint to validate model access.
	return nil
}

func (p *OpenAIProvider) Generate(ctx context.Context, req CompletionRequest) (string, error) {
	openAIReq := openAIChatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
		Tools:    req.Tools,
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := p.HTTPClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp openAIChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != nil {
			return "", fmt.Errorf("openai error: %s", errResp.Error.Message)
		}
		return "", fmt.Errorf("openai returned status %d", resp.StatusCode)
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no choices returned from OpenAI")
}

func (p *OpenAIProvider) GenerateStream(ctx context.Context, req CompletionRequest) (<-chan string, <-chan error) {
	tokenChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(tokenChan)
		defer close(errChan)

		openAIReq := openAIChatRequest{
			Model:    req.Model,
			Messages: req.Messages,
			Stream:   true,
			Tools:    req.Tools,
		}

		body, err := json.Marshal(openAIReq)
		if err != nil {
			errChan <- err
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			errChan <- err
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)

		resp, err := p.HTTPClient.Do(httpReq)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errChan <- fmt.Errorf("openai streaming error: status %d", resp.StatusCode)
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		var collectedToolCalls []ToolCall

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk openAIChatResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				slog.Error("Failed to decode stream chunk", "error", err)
				continue
			}

			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]
				if choice.Delta.Content != "" {
					select {
					case <-ctx.Done():
						return
					case tokenChan <- choice.Delta.Content:
					}
				}

				if len(choice.Delta.ToolCalls) > 0 {
					// Merging tool call chunks is complex for OpenAI as it streams function arguments
					for _, tc := range choice.Delta.ToolCalls {
						if tc.ID != "" {
							collectedToolCalls = append(collectedToolCalls, tc)
						} else if len(collectedToolCalls) > 0 {
							// Append to the last tool call's arguments
							lastIdx := len(collectedToolCalls) - 1
							collectedToolCalls[lastIdx].Function.Arguments = append(collectedToolCalls[lastIdx].Function.Arguments, tc.Function.Arguments...)
						}
					}
				}

				if choice.FinishReason == "tool_calls" && len(collectedToolCalls) > 0 {
					toolCallJSON, _ := json.Marshal(collectedToolCalls)
					select {
					case tokenChan <- fmt.Sprintf("__TOOL_CALL__%s", string(toolCallJSON)):
					default:
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	return tokenChan, errChan
}
