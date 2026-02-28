# Neuron CLI: Phase-Wise Implementation Plan

This document outlines the step-by-step implementation plan for building the Neuron CLI and its embedded Web UI using the 100% Go (Option A) architecture. Each phase builds upon the previous one, ensuring a functional product at every major milestone.

## Phase 1: Core Foundation & CLI Skeleton
**Goal:** Establish the basic CLI structure, command parsing, and the central configuration engine.

*   **Setup:** Initialize `go.mod` (completed) and basic directory structure (completed).
*   **CLI Framework:** Integrate `spf13/cobra` in `cmd/neuron/main.go` to handle command routing.
*   **Config Engine:** Implement `internal/config` using `spf13/viper` to load and validate settings from `~/.config/neuron/config.yaml`.
*   **Basic Commands:** Implement placeholders for `neuron chat`, `neuron run`, `neuron config`, and `neuron ui`.
*   **Logging:** Set up robust structured logging (`slog` or `zap`) to track application state without cluttering the terminal output.

## Phase 2: Local LLM Engine Integration (Ollama)
**Goal:** Enable the CLI to communicate with a local LLM through the Ollama HTTP API.

*   **Provider Interface:** Define a generic `llm.Provider` interface in `internal/llm/provider.go`.
*   **Ollama Adapter:** Implement the interface for Ollama (`internal/llm/ollama.go`), handling model listing, pulling, and chat completions.
*   **Streaming Support:** Ensure the LLM adapter can stream responses chunk-by-chunk.
*   **Basic Terminal Chat:** Connect the `neuron chat` command to the Ollama adapter to allow simple, raw terminal Q&A (no TUI yet).

## Phase 3: The Terminal UI (TUI)
**Goal:** Provide a rich, interactive terminal experience for the chat session.

*   **TUI Framework:** Integrate `charmbracelet/bubbletea` and `lipgloss` in `internal/tui/`.
*   **Chat View:** Build a scrolling chat interface that renders Markdown (using `glow` or similar) and handles streaming token updates gracefully.
*   **Input Handling:** Implement a multi-line input text area for user prompts.
*   **Command Palette (Optional):** Add a quick-action menu (invoked via `/`) within the TUI for commands like `/model`, `/clear`, `/exit`.

## Phase 4: Core Tooling & Sandbox
**Goal:** Give the LLM the ability to interact with the local system safely.

*   **Tool Interface:** Define the tool contract (`internal/tools/tool.go`) requiring execution logic, description, and schema (for the LLM to understand).
*   **Built-in Tools:** 
    *   File System (`read_file`, `write_file`, `list_dir`).
    *   Shell (`run_command`).
*   **LLM Tool Calling:** Update the LLM adapter (Phase 2) to support sending tool definitions to the LLM and handling tool call requests from the LLM.
*   **Sandbox/Policy Engine:** Implement `internal/sandbox`. *Crucially*, require user approval (via a prompt in the TUI) before executing any `shell` or `write` tool calls.

## Phase 5: The Embedded Web UI
**Goal:** Serve a beautiful, modern React application directly from the Go binary.

*   **Vite/React Setup:** Scaffold a React application inside the `ui/` directory using Vite and Tailwind CSS.
*   **Go API Server:** Implement a lightweight HTTP server (using `net/http` or `gin`) in `internal/ui/server.go`.
*   **API Endpoints:** Expose REST/WebSocket endpoints for chat history, sending messages, and streaming responses (reusing the logic from Phase 2/4).
*   **Embedding:** Use Go's `//go:embed` directive to bundle the compiled Vite output (`ui/dist`) into the final Go binary.
*   **The `neuron ui` Command:** Wire the command to start the server and attempt to open the user's default web browser to the local address.

## Phase 6: MCP (Model Context Protocol) Integration
**Goal:** Allow the CLI to connect to external, third-party tool servers.

*   **MCP Client:** Implement an MCP client in `internal/mcp/` capable of spawning external process servers (e.g., `npx @smithery/cli run...`) or connecting to SSE endpoints.
*   **Tool Translation:** Map the discoverable tools from the MCP server into the internal tool format defined in Phase 4.
*   **Lifecycle Management:** Ensure the CLI properly starts, monitors, and cleanly shuts down attached MCP servers.

## Phase 7: Remote Channels (Slack / WhatsApp)
**Goal:** Allow users to chat with their local Neuron agent while away from their computer.

*   **Webhooks:** Add webhook receiver endpoints to the Go HTTP server (from Phase 5).
*   **Adapters:** Implement channel-specific logic in `internal/channels/` to parse incoming messages and format outgoing responses.
*   **Session Management:** Map remote channel threads to specific internal chat sessions to maintain context.

## Phase 8: Polish & Distribution
**Goal:** Prepare the CLI for public release.

*   **Testing:** Expand unit and integration tests, particularly around tool safety and API contracts.
*   **Cross-Compilation:** Configure GoReleaser (`.goreleaser.yaml`) to automatically build binaries for Linux, macOS (Intel/Silicon), and Windows.
*   **Documentation:** Finalize user guides, installation instructions, and configuration references.
