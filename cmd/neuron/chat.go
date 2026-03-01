package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
	"github.com/gaurav3000R/neuron-cli/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with the default model.
This command opens an interactive Terminal UI (TUI) where you can
chat with the model, run tools, and switch models.`,
	Run: func(cmd *cobra.Command, args []string) {
		startInteractiveChat()
	},
}

func startInteractiveChat() {
	baseURL := viper.GetString("llm.base_url")
	model := viper.GetString("llm.default_model")

	provider := llm.NewOllamaProvider(baseURL)
	ctx := context.Background()

	err := provider.Preflight(ctx, model)
	if err != nil {
		fmt.Printf("[Error] Cannot reach Ollama at %s.\n", baseURL)
		fmt.Printf("Make sure Ollama is running (`ollama serve`).\nDetails: %v\n", err)
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	registry.Register(&tools.ReadFileTool{})
	registry.Register(&tools.WriteFileTool{})
	registry.Register(&tools.ShellTool{})

	// Add an example MCP Server connection for Demo (SQLite in this case, a common initial MCP test)
	// You need `npx` installed and an `npx -y @modelcontextprotocol/server-sqlite` command available.
	// In a real application, these paths would be loaded from config.yaml!
	mcpClient, err := tools.LoadMCPServer(ctx, registry, "sqlite", "npx", "-y", "@modelcontextprotocol/server-sqlite", "--help")
	if err != nil {
		fmt.Printf("[Warning] Failed to load example MCP server: %v\n", err)
	} else {
		defer mcpClient.Close()
	}

	if err := tui.Run(provider, model, registry); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
