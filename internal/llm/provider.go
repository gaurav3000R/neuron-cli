package llm

import (
	"context"
	"encoding/json"
)

// MessageRole defines the sender of a chat message.
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// Message represent a single chat message
type Message struct {
	Role       MessageRole `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// CompletionRequest represents the input parameters for an LLM generation.
type CompletionRequest struct {
	Model     string
	Messages  []Message
	Stream    bool
	JSONMode  bool
	Tools     []Definition
}

// Definition represents the schema sent to OpenAI-compatible endpoints or Ollama.
type Definition struct {
	Type     string         `json:"type"`
	Function FunctionSchema `json:"function"`
}

type FunctionSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Provider represents the contract that any integrated LLM backend must fulfill.
type Provider interface {
	// Generate produces a complete response given a conversation history.
	Generate(ctx context.Context, req CompletionRequest) (string, error)

	// GenerateStream produces tokens incrementally over a channel.
	GenerateStream(ctx context.Context, req CompletionRequest) (<-chan string, <-chan error)

	// Preflight checks if the provider is reachable and the model is available.
	Preflight(ctx context.Context, model string) error
}
