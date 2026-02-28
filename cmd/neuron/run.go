package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var prompt string

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
		fmt.Printf("Running headless prompt: '%s' (Implementation Pending Phase 2)\n", prompt)
		// TODO: Pass to raw internal/llm execution mode.
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "The prompt to run headless")
}
