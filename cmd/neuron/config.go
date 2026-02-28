package main

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or edit the CLI configuration",
	Long: `Shows the currently loaded configuration and file location.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Loaded config file: %s\n", viper.ConfigFileUsed())
		fmt.Printf("\nCurrent Configuration Values:\n")
		for _, key := range viper.AllKeys() {
			fmt.Printf("  %s: %v\n", key, viper.Get(key))
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
