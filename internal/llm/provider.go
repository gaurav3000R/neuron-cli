package llm

import (
	"context"
)

// MessageRole defines the sender of a chat message.
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// Message represents a single chat message in the conversation history.
type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

// CompletionRequest represents the input parameters for an LLM generation.
type CompletionRequest struct {
	Model     string
	Messages  []Message
	Stream    bool
	JSONMode  bool
	// Tools will be added in Phase 4
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
