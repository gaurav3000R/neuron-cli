package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	prompt   string
	runModel string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute a prompt blindly (Headless Mode)",
	Long: `Execute a given prompt without entering an interactive chat session.
This is useful for piping input or running CLI scripts.`,
	Run: func(cmd *cobra.Command, args []string) {
		if prompt == "" {
			fmt.Println("Please provide a prompt using the -p flag.")
			return
		}

		model := runModel
		if model == "" {
			// This would be the logic once implemented
			// model = viper.GetString("llm.default_model")
		}

		fmt.Printf("Running headless prompt: '%s' with model '%s' (Implementation Pending Phase 2)\n", prompt, model)
		// TODO: Pass to raw internal/llm execution mode.
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "The prompt to run headless")
	runCmd.Flags().StringVarP(&runModel, "model", "m", "", "The model to use for the headless run")
}
