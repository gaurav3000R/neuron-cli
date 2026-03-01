package channels

import (
	"net/http"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
)

// Channel represents an external remote connection like Slack or WhatsApp
type Channel interface {
	// Name gets the display name of the channel
	Name() string

	// HandleWebhook processes incoming HTTP requests from the platform
	HandleWebhook(w http.ResponseWriter, r *http.Request)

	// SetContext configures the AI capabilities for the channel
	SetContext(provider llm.Provider, modelID string, registry *tools.Registry)
}

// SessionManager stores the ongoing chats mapped by remote platform Thread IDs
type SessionManager struct {
	// Map of channelName:threadID -> []llm.Message
	sessions map[string][]llm.Message
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string][]llm.Message),
	}
}

// GetHistory returns the chat history for a given thread
func (s *SessionManager) GetHistory(threadID string) []llm.Message {
	if history, ok := s.sessions[threadID]; ok {
		return history
	}
	// Start with empty history
	return []llm.Message{}
}

// SaveHistory updates the chat history for a thread
func (s *SessionManager) SaveHistory(threadID string, history []llm.Message) {
	s.sessions[threadID] = history
}

// AppendHistory is a convenience function to append message to a thread's history
func (s *SessionManager) AppendHistory(threadID string, message llm.Message) {
	history := s.GetHistory(threadID)
	history = append(history, message)
	s.SaveHistory(threadID, history)
}
