package channels

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
)

type WhatsAppChannel struct {
	provider llm.Provider
	modelID  string
	registry *tools.Registry
	sessions *SessionManager
}

func NewWhatsAppChannel(sessions *SessionManager) *WhatsAppChannel {
	return &WhatsAppChannel{
		sessions: sessions,
	}
}

func (wa *WhatsAppChannel) Name() string {
	return "whatsapp"
}

func (wa *WhatsAppChannel) SetContext(provider llm.Provider, modelID string, registry *tools.Registry) {
	wa.provider = provider
	wa.modelID = modelID
	wa.registry = registry
}

func (wa *WhatsAppChannel) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// WhatsApp Cloud API sends a GET request for webhook verification
	if r.Method == http.MethodGet {
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		// Verify token (Assume 'NEURON_WA_TOKEN' for real usage)
		if mode == "subscribe" && token != "" {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(challenge))
			return
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Handle POST for actual messages
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

	slog.Info("Received WhatsApp Webhook", "payload", payload)

	// Here a real implementation would extract messages[0].text.body,
	// call wa.provider.Generate, and HTTP POST the result back
	// to graph.facebook.com/vXY.0/{PHONE_ID}/messages

	w.WriteHeader(http.StatusOK)
}
