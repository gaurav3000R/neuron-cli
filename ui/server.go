package ui

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/gaurav3000R/neuron-cli/internal/channels"
	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
)

//go:embed dist/*
var content embed.FS

// StartServer begins the UI HTTP local server and attempts to open a browser window.
func StartServer(port int, provider llm.Provider, registry *tools.Registry, modelID string) error {
	// The Vite build goes into dist/
	// We need to strip the "dist" prefix so the root maps to index.html
	distFS, err := fs.Sub(content, "dist")
	if err != nil {
		return fmt.Errorf("failed to locate embedded UI assets: %w", err)
	}

	http.Handle("/", http.FileServer(http.FS(distFS)))

	// Example hook for where the API endpoints will live
	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok", "version":"1.0.0"}`)
	})

	chatAPI := NewChatServer(provider, modelID, registry)
	http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		chatAPI.HandleChat(w, r)
	})

	// Phase 7: Remote Channel Webhooks
	sessionManager := channels.NewSessionManager()

	slackAdapter := channels.NewSlackChannel(sessionManager)
	slackAdapter.SetContext(provider, modelID, registry)
	http.HandleFunc("/api/webhooks/slack", slackAdapter.HandleWebhook)

	waAdapter := channels.NewWhatsAppChannel(sessionManager)
	waAdapter.SetContext(provider, modelID, registry)
	http.HandleFunc("/api/webhooks/whatsapp", waAdapter.HandleWebhook)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	slog.Info("Starting Web UI Server", "url", "http://"+addr)

	// Automatically open the browser
	openBrowser("http://" + addr)

	return http.ListenAndServe(addr, nil)
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}

	if err != nil {
		slog.Warn("Failed to automatically open browser", "error", err)
	}
}
