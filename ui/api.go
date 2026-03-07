package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/skills"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
	slog.Info("Incoming chat request", "remote_addr", r.RemoteAddr, "method", r.Method)
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

	sysCtx := skills.LoadContext()
	if sysCtx != "" {
		req.Messages = append([]llm.Message{{Role: llm.RoleSystem, Content: sysCtx}}, req.Messages...)
	}

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
	var process func([]llm.Message, int)

	process = func(history []llm.Message, depth int) {
		// Prevent infinite loops - max 3 tool calls per request
		if depth > 3 {
			slog.Warn("Max tool call depth reached, stopping", "depth", depth)
			sendEvent("\n⚠️ Maximum tool execution limit reached. Please refine your question.\n", "")
			return
		}

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

		// Disable tools after first execution to force text response
		if depth > 0 {
			llmTools = nil
			slog.Info("Disabling tools to force text response", "depth", depth)
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
					// Stream closed, check for errors
					select {
					case err := <-errChan:
						if err != nil {
							slog.Error("Stream error after close", "error", err)
							sendEvent("", err.Error())
						}
					default:
					}
					slog.Info("Token channel closed, stream complete")
					return
				}

				// Debug: show first 20 chars
				prefix := token
				if len(token) > 20 {
					prefix = token[:20]
				}
				slog.Info("Web UI received token", "token_prefix", prefix, "length", len(token))

				// Check for tool call marker using strings.HasPrefix
				if strings.HasPrefix(token, "__TOOL_CALL__") {
					slog.Info("Web UI detected tool call marker - processing")
					toolCallJSON := strings.TrimPrefix(token, "__TOOL_CALL__")
					slog.Debug("Tool call JSON", "json", toolCallJSON)

					var toolCalls []llm.ToolCall
					if err := json.Unmarshal([]byte(toolCallJSON), &toolCalls); err != nil {
						slog.Error("Failed to parse tool calls", "error", err, "json", toolCallJSON)
						sendEvent(fmt.Sprintf("Error parsing tool calls: %v", err), "")
						return
					}

					if len(toolCalls) > 0 {
						// Execute tool
						tc := toolCalls[0]
						tool, ok := s.registry.Get(tc.Function.Name)
						if ok {
							slog.Info("Executing tool", "name", tc.Function.Name, "args", string(tc.Function.Arguments))
							sendEvent(fmt.Sprintf("🔧 %s...\n", tc.Function.Name), "")
							result, err := tool.Execute(ctx, tc.Function.Arguments)
							if err != nil {
								slog.Error("Tool execution failed", "name", tc.Function.Name, "error", err)
								sendEvent(fmt.Sprintf("❌ Error: %v\n\n", err), "")
								// Add error to history and continue with text response
								history = append(history, llm.Message{
									Role:      llm.RoleAssistant,
									Content:   "",
									ToolCalls: toolCalls,
								})
								history = append(history, llm.Message{
									Role:    llm.RoleUser,
									Content: fmt.Sprintf("[Tool Error]\n%v", err),
								})
								// Continue with text response (no tools)
								process(history, depth+1)
								return
							} else {
								slog.Info("Tool execution successful", "name", tc.Function.Name, "result_len", len(result))
								// Don't show raw result to user - let the model explain it
								// Add tool result to history and continue
								history = append(history, llm.Message{
									Role:      llm.RoleAssistant,
									Content:   "",
									ToolCalls: toolCalls,
								})
								history = append(history, llm.Message{
									Role:    llm.RoleUser,
									Content: fmt.Sprintf("[Tool Result]\n%s", result),
								})
								// Recursively process with tool result
								process(history, depth+1)
								return
							}
						} else {
							slog.Error("Tool not found", "tool", tc.Function.Name)
							sendEvent(fmt.Sprintf("❌ Tool '%s' not found\n", tc.Function.Name), "")
							// Continue anyway
							return
						}
					}
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

	process(messages, 0)
}
