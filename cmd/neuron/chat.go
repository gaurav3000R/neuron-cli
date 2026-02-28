package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with the default model.
This command opens an interactive Terminal UI (TUI) where you can
chat with the model, run tools, and switch models.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Interactive chat session starting... (Implementation Pending Phase 3)")
		// TODO: Call internal/tui Run logic here.
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
