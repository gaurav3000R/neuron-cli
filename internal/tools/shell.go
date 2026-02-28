package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ShellTool allows the LLM to execute bash/sh commands on the host system.
type ShellTool struct{}

func (t *ShellTool) Name() string {
	return "run_command"
}

func (t *ShellTool) Description() string {
	return "Executes a shell command (bash/sh) on the local machine and returns the standard output/error."
}

func (t *ShellTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The shell command to execute"
			}
		},
		"required": ["command"]
	}`)
}

func (t *ShellTool) RequiresApproval() bool {
	return true // Shell execution is inherently dangerous
}

func (t *ShellTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments for run_command: %w", err)
	}

	// For simplicity, we execute via bash. In a cross-platform implementation,
	// we'd switch to cmd.exe or powershell on Windows.
	cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	
	// Combine outputs to give context back to the LLM
	result := stdout.String()
	if stderr.Len() > 0 {
		result += "\n[STDERR]\n" + stderr.String()
	}

	if err != nil {
		return fmt.Sprintf("Command failed with error: %v\nOutput: %s", err, result), nil
	}

	return result, nil
}
