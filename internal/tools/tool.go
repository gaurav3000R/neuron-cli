package tools

import (
	"context"
	"encoding/json"
)

// Tool defines the contract for any executable capability provided to the LLM.
type Tool interface {
	// Name must match the schema name provided to the LLM
	Name() string
	
	// Description explains what the tool does (used by LLM)
	Description() string
	
	// Schema returns the JSON schema representing the expected parameters
	Schema() json.RawMessage
	
	// Execute performs the actual work given the JSON arguments from the LLM
	Execute(ctx context.Context, args json.RawMessage) (string, error)
	
	// RequiresApproval determines if the Sandbox must prompt the user first
	RequiresApproval() bool
}

// Registry manages the collection of available tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry initializes an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get finds a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// GetAll returns all registered tools.
func (r *Registry) GetAll() []Tool {
	all := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		all = append(all, t)
	}
	return all
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

// GetDefinitions returns the schema format expected by standard LLM tool calling APIs.
func (r *Registry) GetDefinitions() []Definition {
	defs := make([]Definition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, Definition{
			Type: "function",
			Function: FunctionSchema{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Schema(),
			},
		})
	}
	return defs
}
