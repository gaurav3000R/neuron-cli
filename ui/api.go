package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
)

type ChatServer struct {
	provider llm.Provider
	modelID  string
	registry *tools.Registry
}

func NewChatServer(provider llm.Provider, modelID string, registry *tools.Registry) *ChatServer {
	return &ChatServer{
		provider: provider,
		modelID:  modelID,
		registry: registry,
	}
}

type ChatRequest struct {
	Messages []llm.Message `json:"messages"`
}

type SSEEvent struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

func (s *ChatServer) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Setup SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Allow CORS for local dev
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	s.runChatStream(ctx, w, flusher, req.Messages)
}

func (s *ChatServer) runChatStream(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, messages []llm.Message) {
	sendEvent := func(content string, errorMsg string) {
		event := SSEEvent{Content: content, Error: errorMsg}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Recursive function pattern for tool calls
	var process func([]llm.Message)

	process = func(history []llm.Message) {
		// Use definitions
		toolDefs := s.registry.GetDefinitions()
		var llmTools []llm.Definition
		for _, def := range toolDefs {
			llmTools = append(llmTools, llm.Definition{
				Type: def.Type,
				Function: llm.FunctionSchema{
					Name:        def.Function.Name,
					Description: def.Function.Description,
					Parameters:  def.Function.Parameters,
				},
			})
		}

		compReq := llm.CompletionRequest{
			Model:    s.modelID,
			Messages: history,
			Stream:   true,
			Tools:    llmTools,
		}

		tokenChan, errChan := s.provider.GenerateStream(ctx, compReq)
		var currentGen string

		for {
			select {
			case <-ctx.Done():
				return
			case token, ok := <-tokenChan:
				if !ok {
					// Stream closed
					// Check error
					err := <-errChan
					if err != nil {
						sendEvent("", err.Error())
						return
					}

					// We need to parse if this was a tool call.
					// In our Ollama implementation, tool calls might not be cleanly parsed yet
					// if they were just streamed. But for simplicity, we assume text stream.
					return
				}

				// Standard token
				currentGen += token
				sendEvent(token, "")

			case err := <-errChan:
				if err != nil {
					sendEvent("", err.Error())
				}
				return
			}
		}
	}

	process(messages)
}
