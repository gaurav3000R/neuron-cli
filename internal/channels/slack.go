package channels

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
)

type SlackChannel struct {
	provider llm.Provider
	modelID  string
	registry *tools.Registry
	sessions *SessionManager
}

func NewSlackChannel(sessions *SessionManager) *SlackChannel {
	return &SlackChannel{
		sessions: sessions,
	}
}

func (s *SlackChannel) Name() string {
	return "slack"
}

func (s *SlackChannel) SetContext(provider llm.Provider, modelID string, registry *tools.Registry) {
	s.provider = provider
	s.modelID = modelID
	s.registry = registry
}

func (s *SlackChannel) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Handle Slack URL Verification
	if eventType, ok := payload["type"].(string); ok && eventType == "url_verification" {
		challenge := payload["challenge"].(string)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(challenge))
		return
	}

	slog.Info("Received Slack Webhook payload", "payload", payload)

	// Here a real implementation would extract text, sender, and thread_ts,
	// parse it through s.provider.Generate, and HTTP POST the result back
	// to chat.postMessage using a Slack bot token.

	// Example thread tracking:
	// threadID := fmt.Sprintf("slack:%s", event.ThreadTS)
	// s.sessions.AppendHistory(threadID, llm.Message{Role: llm.RoleUser, Content: event.Text})
	// req := llm.CompletionRequest{Messages: s.sessions.GetHistory(threadID), ...}
	// ... provider.Generate(...)
	// s.sessions.AppendHistory(threadID, llm.Message{Role: llm.RoleAssistant, Content: reply})

	w.WriteHeader(http.StatusOK)
}
