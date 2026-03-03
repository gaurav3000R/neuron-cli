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

var modelFlag string

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
	model := modelFlag
	if model == "" {
		model = viper.GetString("llm.default_model")
	}

	provider := llm.NewOllamaProvider(baseURL)
	ctx := context.Background()

	err := provider.Preflight(ctx, model)
	if err != nil {
		fmt.Printf("❌ Preflight check failed for model %q at %s\n", model, baseURL)
		fmt.Printf("Details: %v\n", err)
		fmt.Printf("\n💡 Make sure Ollama is running: ollama serve\n")
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	registry.Register(&tools.ReadFileTool{})
	registry.Register(&tools.WriteFileTool{})
	registry.Register(&tools.ShellTool{})

	if err := tui.Run(provider, model, registry); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "The model to use for the chat session")
}
