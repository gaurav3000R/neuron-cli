# Neuron CLI Blueprint

This blueprint is designed for a developer-first AI CLI. It prioritizes optimal distribution speed, native multi-platform compatibility, minimal user disruption, and a streamlined developer experience. It favors a **Single Binary Architecture**.

## 1. Product Goal

Build a developer-first AI CLI that supports:

- Local and cloud model routing
- Code-aware tools (search, read, edit, run, git)
- MCP-based tool/plugin ecosystem
- Policy-gated execution with sandbox options
- Interactive TUI and non-interactive automation mode
- **Web-based UI** accessible locally alongside the CLI
- Remote access via chat channels (Slack/WhatsApp)
- Zero-dependency cross-platform distribution (Linux/macOS/Windows)

## 2. Recommended Tech Stack

The architecture must yield a single, dependency-free binary that boots in under 50ms and seamlessly leverages multi-core or GPU constraints on the user's local instance. To support the **Web UI**, the frontend application will be compiled into the final binary as static assets.

### 2.1 Option A: The "Ollama/GitHub" Architecture (100% Go) - *Highly Recommended*

If ultimate speed, reliable offline local-llm inference, and broad system-level resource access (GPU memory management, shell processes) are the highest priorities:

- **Language:** Go 1.22+
- **CLI Framework:** `spf13/cobra` or `urfave/cli/v2`
- **TUI Engine:** `charmbracelet/bubbletea` + `lipgloss` (Unparalleled beautiful terminal UI)
- **Web UI:** React/Vite frontend embedded natively via `go:embed`. Served locally via Go HTTP standard library or `gin`. Both the CLI and UI use the exact same internal Go APIs.
- **Local LLM Engine:** Native Go bindings over `llama.cpp` (e.g. `go-llama.cpp`) or via `ollama` API.
- **Config & Validation:** `spf13/viper`, `go-playground/validator`
- **Build & Release:** `goreleaser` (Automated building for all OS/Arch combinations)
- **Database:** `mattn/go-sqlite3` or `glebarez/sqlite` (pure Go SQLite)

**Why Go?**
Go is the industry standard for cloud-native CLI tooling (Docker, Kubernetes, GitHub CLI, Ollama). Using `go:embed`, you can compile a full 10MB React Single Page Application (SPA) directly into the binary with zero performance hit or extra files for the user to manage.

### 2.2 Option B: The "Modern Web" Architecture (100% TypeScript + Bun)

If the team has exclusive TypeScript expertise, needs to leverage vast NPM ecosystems (like MCP SDKs natively), and prioritizes UI development speed:

- **Runtime Target:** Bun (Compiles TS to a single executable via `bun build --compile`)
- **Language:** TypeScript (Strict Mode)
- **CLI Framework:** `commander` or `cleye`
- **TUI Engine:** `ink` (React for the terminal)
- **Web UI:** React/Vite/NextJS API routes served by Bun's native lightning-fast `Bun.serve`. Compiled React frontend assets embedded via Bun loaders.
- **Validation:** `zod`
- **Database:** `libSQL` or `bun:sqlite`
- **Local LLM Engine:** `node-llama-cpp` (pre-compiled bindings)

**Why Bun/TS?**
Node.js is extremely difficult to distribute as a single binary without awful UX (like pkg or Nexe injecting massive node virtual instances). Bun compiles pure TypeScript apps directly into native executables. You can build both the terminal app and Web API in unified TS.

---

*(Note: The rest of the blueprint assumes Option A or B is chosen, establishing a single-language monorepo rather than a multi-language microservice architecture).*

## 3. Monorepo Structure

```txt
neuron-cli/
├── cmd/
│   └── neuron/                # The main entrypoint
├── internal/                  # Private application code
│   ├── ui/                    # Web API servers bridging models to web client
│   ├── tui/                   # Terminal UI views (Bubbletea or Ink)
│   ├── cli/                   # Command parsing, flag handling
│   ├── config/                # Settings, policies, state validation
│   ├── llm/                   # Model routing, Ollama/llama.cpp adapters, API routing
│   ├── mcp/                   # Model Context Protocol server/client orchestration
│   ├── channels/              # Slack/WhatsApp adapters
│   ├── tools/                 # Built-in file, search, shell tools
│   └── sandbox/               # Execution environments (none, docker)
├── ui/                        # **NEW**: The React/Vite Frontend Application Directory
├── pkg/                       # Public SDK code (if building extensions)
├── docs/                      # Architecture, tools, docs
├── schemas/                   # JSON schemas for settings
├── Makefile / package.json    # Build scripts
└── .goreleaser.yaml           # Release automation (Option A only)
```

## 4. Core Subsystems

### 4.1 Command Surface

Base commands:
- `neuron chat` (Default interactive terminal session)
- `neuron root` / `neuron ui` (**NEW**: Starts the Web Server and opens `localhost:3133` in the browser)
- `neuron run -p "..."` (Headless agent mode)
- `neuron model` (List, switch remote vs local)
- `neuron tools` (Enable, disable built-in/MCP tools)
- `neuron config` (View/edit settings)
- `neuron channels` (Slack/WhatsApp setup)

### 4.2 Handling the Web UI + CLI Concurrency

The system should run a background daemon (or run API endpoints internally when the UI is booted) so both the CLI and Web UI share the identical state. 

1. **State:** Both the CLI and Web interface pull conversation history from `~/.config/neuron/data.db`.
2. **API:** The UI makes HTTP or WebSocket requests to the local port (e.g. `http://127.0.0.1:3133/api/chat`).
3. **Execution:** The backend logic mapping tool calls, routing, and approvals belongs exclusively to `internal/llm` and `internal/tools`. The CLI/TUI and Web UI merely submit intents to this internal Go/Bun logic.

### 4.3 Tooling System

Built-in tools to ship immediately (Fast, Native):
- **fs:** fast read/write/list/glob
- **search:** ripgrep wrapper
- **shell:** Safe execution with Approval Prompts (sent via terminal or websocket to web UI)
- **git:** Status, diff, commit helpers

### 4.4 Model and Provider Layer

**Local-First Architecture:**
1. **Ollama Integration:** Fast-path to `http://localhost:11434`. (Primary local provider)
2. **Cloud Fallback:** Anthropic/OpenAI if `preferLocal=false`. 

### 4.5 Persistence

- Store session history, tool logs, and cached context in a local SQLite file (`~/.config/neuron/data.db`).

## 5. Implementation Milestones

### Milestone 0: Foundation (Week 1)
- Single Binary Monorepo scaffolding
- Config loader + schema validation
- Basic `neuron --help` and routing skeleton

### Milestone 1: Core API & Terminal CLI (Week 2)
- TUI app shell (Bubbletea or Ink)
- Core `llm` and `tools` internal packages (Streaming, Tool calling logic)

### Milestone 2: The Web UI (Week 3) (NEW)
- Scaffold Vite/React app in `/ui`
- Bind API server in core to serve SPA via `go:embed` or Bun bundler
- Implement Chat, Config, and Tool Toggle screens in React
- Connect Web UI websockets to Core API logs. 

### Milestone 3: Local Engine & Tools (Week 4)
- Ollama local integration
- Local Web UI/TUI approval interceptors (prompts for `shell` allow/deny).

### Milestone 4: MCP + Extensions + Channels (Week 5 & 6)
- MCP client process spawning
- Slack/WhatsApp webhooks mapping

### Milestone 5: Distribution (Week 7)
- CI Matrix (Linux/macOS/Windows)
- Compile CLI + embedded React UI into a single `.exe`, `.tar.gz`, via GitHub Actions.
