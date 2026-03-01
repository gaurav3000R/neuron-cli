package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaurav3000R/neuron-cli/internal/logger"
	"github.com/spf13/viper"
)

// AppConfig represents the root configuration structure.
type AppConfig struct {
	LLM struct {
		Provider     string `mapstructure:"provider"` // e.g., "ollama", "openai"
		DefaultModel string `mapstructure:"default_model"`
		BaseURL      string `mapstructure:"base_url"` // e.g., "http://localhost:11434"
		PreferLocal  bool   `mapstructure:"prefer_local"`
	} `mapstructure:"llm"`

	Sandbox struct {
		Policy string `mapstructure:"policy"` // "default", "auto_edit", "plan"
		Type   string `mapstructure:"type"`   // "none", "docker"
	} `mapstructure:"sandbox"`
}

// Config holds the decoded configuration singleton.
var C AppConfig

// InitConfig initializes viper and loads the configuration file.
func InitConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Failed to get user home directory", "error", err)
		os.Exit(1)
	}

	configDir := filepath.Join(home, ".config", "neuron")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		slog.Error("Failed to create config directory", "error", err)
		os.Exit(1)
	}

	viper.AddConfigPath(configDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Set Defaults
	viper.SetDefault("llm.provider", "ollama")
	viper.SetDefault("llm.default_model", "llama3")
	viper.SetDefault("llm.base_url", "http://localhost:11434")
	viper.SetDefault("llm.prefer_local", true)

	viper.SetDefault("sandbox.policy", "default")
	viper.SetDefault("sandbox.type", "none")

	// Environment variable overrides
	viper.SetEnvPrefix("NEURON")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; generate default
			configPath := filepath.Join(configDir, "config.yaml")
			if err := viper.SafeWriteConfigAs(configPath); err != nil {
				slog.Warn("Failed to write default config file", "error", err)
			} else {
				slog.Info("Created default configuration file", "path", configPath)
			}
		} else {
			slog.Error("Error reading config file", "error", err)
		}
	}

	if err := viper.Unmarshal(&C); err != nil {
		slog.Error("Unable to decode configuration into struct", "error", err)
		os.Exit(1)
	}
}

// SetupLogger configures the default slog logger using the new logger module.
func SetupLogger(verbose bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	logDir := filepath.Join(home, ".config", "neuron")
	jsonLogFile := filepath.Join(logDir, "neuron.json.log")
	prettyLogFile := filepath.Join(logDir, "neuron.log")

	// 1. Setup JSON logging to file for deep micro-debugging
	// 2. Setup Pretty (Colorized) logging to file only (NO terminal output to avoid breaking TUI)
	err = logger.SetupMulti(verbose, jsonLogFile, prettyLogFile, nil)
	if err != nil {
		// Fallback to basic logging if setup fails
		// Don't use slog.Error here as it might not be initialized yet
	}
}
