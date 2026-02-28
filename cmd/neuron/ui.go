package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the embedded Web UI",
	Long: `Start the local Web UI server and open it in the default browser.
This allows you to chat with Neuron using a graphical interface instead 
of the terminal.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting Web UI server on localhost:3133... (Implementation Pending Phase 5)")
		// TODO: Call internal/ui server struct running logic.
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
