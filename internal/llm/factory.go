package llm

import (
	"os"
	"strings"
)

// NewProvider is a factory function that returns the appropriate LLM provider
// based on the configuration.
func NewProvider(providerName, baseURL, apiKey string) Provider {
	providerName = strings.ToLower(providerName)

	switch providerName {
	case "openai":
		if baseURL == "" || baseURL == "http://localhost:11434" {
			baseURL = "https://api.openai.com/v1"
		}
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		return NewOpenAIProvider(baseURL, apiKey)
	case "gemini":
		if baseURL == "" || baseURL == "http://localhost:11434" {
			// Gemini's OpenAI-compatible endpoint
			baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
		}
		if apiKey == "" {
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
		return NewOpenAIProvider(baseURL, apiKey)
	case "huggingface", "hf":
		if baseURL == "" || baseURL == "http://localhost:11434" {
			// HuggingFace's OpenAI-compatible endpoint for serverless inference
			baseURL = "https://api-inference.huggingface.co/v1"
		}
		if apiKey == "" {
			apiKey = os.Getenv("HUGGINGFACE_API_KEY")
			if apiKey == "" {
				apiKey = os.Getenv("HF_TOKEN")
			}
		}
		return NewOpenAIProvider(baseURL, apiKey)
	case "ollama":
		fallthrough
	default: // default to ollama
		return NewOllamaProvider(baseURL)
	}
}
