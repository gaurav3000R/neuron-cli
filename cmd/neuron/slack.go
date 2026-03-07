package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gaurav3000R/neuron-cli/internal/channels"
	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var slackModelFlag string
var slackProviderFlag string

var slackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Start the Slack Bot",
	Long: `Start the Neuron Slack bot in Socket Mode.
Requires SLACK_BOT_TOKEN and SLACK_APP_TOKEN to be set in the environment or .env file.
`,
	Run: func(cmd *cobra.Command, args []string) {
		startSlackBot()
	},
}

func init() {
	rootCmd.AddCommand(slackCmd)
	slackCmd.Flags().StringVarP(&slackModelFlag, "model", "m", "", "The model to use")
	slackCmd.Flags().StringVarP(&slackProviderFlag, "provider", "p", "", "The provider to use")
}

func startSlackBot() {
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")

	if botToken == "" || appToken == "" {
		fmt.Println("❌ Missing SLACK_BOT_TOKEN or SLACK_APP_TOKEN environment variables")
		fmt.Println("💡 Add them to your .env file or export them in your shell.")
		os.Exit(1)
	}

	baseURL := viper.GetString("llm.base_url")

	model := slackModelFlag
	if model == "" {
		model = viper.GetString("llm.default_model")
	}

	providerName := slackProviderFlag
	if providerName == "" {
		providerName = viper.GetString("llm.provider")
	}
	apiKey := viper.GetString("llm.api_key")

	provider := llm.NewProvider(providerName, baseURL, apiKey)
	ctx := context.Background()

	err := provider.Preflight(ctx, model)
	if err != nil {
		fmt.Printf("❌ Preflight check failed for model %q\nDetails: %v\n", model, err)
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	registry.Register(&tools.ReadFileTool{})
	registry.Register(&tools.WriteFileTool{})
	registry.Register(&tools.ShellTool{})

	sessionManager := channels.NewSessionManager()
	slackChannel := channels.NewSlackChannel(sessionManager)
	slackChannel.SetContext(provider, model, registry)

	fmt.Printf("🚀 Starting Neuron Slack Bot (Model: %s)\n", model)
	if err := slackChannel.RunSocketMode(ctx, botToken, appToken); err != nil {
		fmt.Printf("❌ Slack Bot crashed: %v\n", err)
		os.Exit(1)
	}
}
