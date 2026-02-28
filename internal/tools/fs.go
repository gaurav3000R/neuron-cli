package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// ReadFileTool allows the LLM to read local file contents.
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Reads the contents of a local file at the given absolute or relative path."
}

func (t *ReadFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to the file to read"
			}
		},
		"required": ["path"]
	}`)
}

func (t *ReadFileTool) RequiresApproval() bool {
	// Reading files usually doesn't require approval in default policy,
	// though strict sandboxes might toggle this.
	return false
}

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments for read_file: %w", err)
	}

	data, err := os.ReadFile(params.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Truncate massive files to avoid context blowout (in a real app, this would be configurable).
	// Limit to ~20k characters for safety.
	content := string(data)
	if len(content) > 20000 {
		content = content[:20000] + "\n...[TRUNCATED]"
	}

	return content, nil
}

// WriteFileTool allows the LLM to write or overwrite local files.
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Creates or overwrites a file with the provided content."
}

func (t *WriteFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to the file to write"
			},
			"content": {
				"type": "string",
				"description": "The complete content to write to the file"
			}
		},
		"required": ["path", "content"]
	}`)
}

func (t *WriteFileTool) RequiresApproval() bool {
	return true // Writing files is a mutating action, must be approved.
}

func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments for write_file: %w", err)
	}

	if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(params.Content), params.Path), nil
}
