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

var uiModelFlag string

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the embedded Web UI",
	Long: `Start the local Web UI server and open it in the default browser.
This allows you to chat with Neuron using a graphical interface instead 
of the terminal.`,
	Run: func(cmd *cobra.Command, args []string) {
		baseURL := viper.GetString("llm.base_url")
		model := uiModelFlag
		if model == "" {
			model = viper.GetString("llm.default_model")
		}

		provider := llm.NewOllamaProvider(baseURL)
		ctx := context.Background()

		err := provider.Preflight(ctx, model)
		if err != nil {
			fmt.Printf("❌ Preflight check failed for model %q at %s\n", model, baseURL)
			fmt.Printf("Details: %v\n", err)
			os.Exit(1)
		}

		registry := tools.NewRegistry()
		registry.Register(&tools.ReadFileTool{})
		registry.Register(&tools.WriteFileTool{})
		registry.Register(&tools.ShellTool{})

		port := 3133
		fmt.Printf("🚀 Starting Neuron Web UI on http://localhost:%d\n", port)
		fmt.Printf("💡 Press Ctrl+C to stop the server\n\n")

		if err := ui.StartServer(port, provider, registry, model); err != nil {
			if err.Error() == "listen tcp 127.0.0.1:3133: bind: address already in use" ||
				err.Error() == "listen tcp [::1]:3133: bind: address already in use" {
				fmt.Printf("❌ Port %d is already in use\n", port)
				fmt.Printf("💡 Kill the existing process with: lsof -ti:%d | xargs kill -9\n", port)
				fmt.Printf("   Or run: npm run kill\n")
			} else {
				fmt.Printf("❌ Web Server crashed: %v\n", err)
			}
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
	uiCmd.Flags().StringVarP(&uiModelFlag, "model", "m", "", "The model to use for the UI session")
}
