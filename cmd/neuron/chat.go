package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tui"
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

	if err := tui.Run(provider, model); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
