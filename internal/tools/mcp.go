package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPAdapter implements the Tool interface by proxying calls to an MCP server.
type MCPAdapter struct {
	client     *client.Client
	serverName string
	toolDef    mcp.Tool
}

// Name returns the tool name as defined by the MCP server, optionally prefixed by the server name to avoid collisions.
func (a *MCPAdapter) Name() string {
	return fmt.Sprintf("%s_%s", a.serverName, a.toolDef.Name)
}

// Description returns the tool description from the MCP server.
func (a *MCPAdapter) Description() string {
	return a.toolDef.Description
}

// Schema passes through the JSON schema defined by the MCP server.
func (a *MCPAdapter) Schema() json.RawMessage {

	// The mcp-go library returns parameters as a generic interface{}.
	// We need to marshal it back to raw JSON for Ollama to understand.
	b, err := json.Marshal(a.toolDef.InputSchema)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(b)
}

// RequiresApproval determines if the Sandbox should prompt.
// By default, we treat ALL external MCP tools as requiring sandbox approval for safety.
func (a *MCPAdapter) RequiresApproval() bool {
	return true
}

// Execute proxies the tool execution payload over stdio to the MCP server.
func (a *MCPAdapter) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	slog.Info("Executing MCP Tool", "server", a.serverName, "tool", a.toolDef.Name)

	var parsedArgs map[string]interface{}
	if err := json.Unmarshal(args, &parsedArgs); err != nil {
		return "", fmt.Errorf("failed to parse MCP tool args: %w", err)
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = a.toolDef.Name
	req.Params.Arguments = parsedArgs

	res, err := a.client.CallTool(ctx, req)
	if err != nil {
		return "", fmt.Errorf("MCP execution failed: %w", err)
	}

	if res.IsError {
		return "", fmt.Errorf("MCP server reported error during execution")
	}

	// MCP responses can be complex (mixed text and images).
	// For CLI LLM usage, we stringify the text contents.
	var finalResult string
	for _, content := range res.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			finalResult += textContent.Text + "\n"
		} else {
			finalResult += "[Unsupported Media Content Returned]\n"
		}
	}

	return finalResult, nil
}

// LoadMCPServer starts a new stdio MCP client and registers all its tools to the local registry.
func LoadMCPServer(ctx context.Context, registry *Registry, serverName string, cmd string, args ...string) (*client.Client, error) {

	// Create the standard I/O client
	stdioTransport := transport.NewStdio(cmd, nil, args...)
	mcpClient := client.NewClient(stdioTransport)

	if err := mcpClient.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start MCP process %s: %w", serverName, err)
	}

	// Initialize the protocol handshake
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "neuron-cli",
		Version: "1.0.0",
	}

	_, err := mcpClient.Initialize(ctx, initReq)
	if err != nil {
		mcpClient.Close()
		return nil, fmt.Errorf("MCP handshake failed for %s: %w", serverName, err)
	}

	// Fetch all tools hosted by this server
	toolsRes, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		mcpClient.Close()
		return nil, fmt.Errorf("failed to list tools for %s: %w", serverName, err)
	}

	// Register each MCP tool as a local adapter into our main LLM Tool Registry
	count := 0
	for _, t := range toolsRes.Tools {
		adapter := &MCPAdapter{
			client:     mcpClient,
			serverName: serverName,
			toolDef:    t,
		}
		registry.Register(adapter)
		count++
	}

	slog.Info("Loaded MCP Server", "name", serverName, "tools_count", count)

	return mcpClient, nil
}
