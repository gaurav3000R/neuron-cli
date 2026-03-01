package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
	"github.com/gaurav3000R/neuron-cli/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the embedded Web UI",
	Long: `Start the local Web UI server and open it in the default browser.
This allows you to chat with Neuron using a graphical interface instead 
of the terminal.`,
	Run: func(cmd *cobra.Command, args []string) {
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

		port := 3133
		if err := ui.StartServer(port, provider, registry, model); err != nil {
			fmt.Printf("Web Server crashed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
