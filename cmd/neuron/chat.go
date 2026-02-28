package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/gaurav3000R/neuron-cli/internal/llm"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with the default model.
This command opens an interactive Terminal UI (TUI) where you can
chat with the model, run tools, and switch models.`,
	Run: func(cmd *cobra.Command, args []string) {
		startInteractiveChat()
	},
}

func startInteractiveChat() {
	baseURL := viper.GetString("llm.base_url")
	model := viper.GetString("llm.default_model")
	
	fmt.Printf("Starting Neuron Chat (%s)\n", model)
	fmt.Println("Type '/exit' or Ctrl+C to quit.")
	fmt.Println("------------------------------------------------")

	provider := llm.NewOllamaProvider(baseURL)

	ctx := context.Background()

	err := provider.Preflight(ctx, model)
	if err != nil {
		fmt.Printf("\n[Error] Cannot reach Ollama at %s.\n", baseURL)
		fmt.Printf("Make sure Ollama is running (`ollama serve`).\nDetails: %v\n", err)
		os.Exit(1)
	}

	var history []llm.Message
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		
		if input == "/exit" || input == "/quit" {
			break
		}

		history = append(history, llm.Message{
			Role:    llm.RoleUser,
			Content: input,
		})

		req := llm.CompletionRequest{
			Model:    model,
			Messages: history,
			Stream:   true,
		}

		tokenChan, errChan := provider.GenerateStream(ctx, req)

		var assistantResponse strings.Builder
		fmt.Print("\nNeuron: ")

		// Stream tokens to stdout
		for token := range tokenChan {
			fmt.Print(token)
			assistantResponse.WriteString(token)
		}

		// Check if the stream ended with an error
		if err := <-errChan; err != nil {
			fmt.Printf("\n[Error during generation: %v]\n", err)
			continue
		}

		fmt.Println() // Newline after complete response

		history = append(history, llm.Message{
			Role:    llm.RoleAssistant,
			Content: assistantResponse.String(),
		})
	}
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
