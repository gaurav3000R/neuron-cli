package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/gaurav3000R/neuron-cli/internal/config"
)

var (
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "neuron",
	Short: "A developer-first AI CLI",
	Long: `Neuron is an open-source, local-first AI CLI for developers.
It supports local and cloud models, tool execution, and an embedded Web UI.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() string {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return ""
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	config.SetupLogger(verbose)
	config.InitConfig()
}
