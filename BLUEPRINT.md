# Neuron CLI Blueprint

This blueprint is designed for a developer-first AI CLI. It prioritizes optimal distribution speed, native multi-platform compatibility, minimal user disruption, and a streamlined developer experience. It favors a **Single Binary Architecture**.

## 1. Product Goal

Build a developer-first AI CLI that supports:

- Local and cloud model routing
- Code-aware tools (search, read, edit, run, git)
- MCP-based tool/plugin ecosystem
- Policy-gated execution with sandbox options
- Interactive TUI and non-interactive automation mode
- Remote access via chat channels (Slack/WhatsApp)
- Zero-dependency cross-platform distribution (Linux/macOS/Windows)

## 2. Recommended Tech Stack

The architecture must yield a single, dependency-free binary that boots in under 50ms and seamlessly leverages multi-core or GPU constraints on the user's local instance.

### 2.1 Option A: The "Ollama/GitHub" Architecture (100% Go) - *Highly Recommended*

If ultimate speed, reliable offline local-llm inference, and broad system-level resource access (GPU memory management, shell processes) are the highest priorities:

- **Language:** Go 1.22+
- **CLI Framework:** `spf13/cobra` or `urfave/cli/v2`
- **TUI Engine:** `charmbracelet/bubbletea` + `lipgloss` (Unparalleled beautiful terminal UI)
- **Local LLM Engine:** Native Go bindings over `llama.cpp` (e.g. `go-llama.cpp`) or via `ollama` API.
- **Config & Validation:** `spf13/viper`, `go-playground/validator`
- **Build & Release:** `goreleaser` (Automated building for all OS/Arch combinations)
- **Database:** `mattn/go-sqlite3` or `glebarez/sqlite` (pure Go SQLite)

**Why Go?**
Go is the industry standard for cloud-native CLI tooling (Docker, Kubernetes, GitHub CLI, Ollama). It provides native concurrency (goroutines), effortless cross-compilation, a statically linked single binary (no Node/Python installed locally), and millisecond boot times. 

### 2.2 Option B: The "Modern Web" Architecture (100% TypeScript + Bun)

If the team has exclusive TypeScript expertise, needs to leverage vast NPM ecosystems (like MCP SDKs natively), and prioritizes UI development speed:

- **Runtime Target:** Bun (Compiles TS to a single executable via `bun build --compile`)
- **Language:** TypeScript (Strict Mode)
- **CLI Framework:** `commander` or `cleye`
- **TUI Engine:** `ink` (React for the terminal)
- **Validation:** `zod`
- **Database:** `libSQL` or `bun:sqlite`
- **Local LLM Engine:** `node-llama-cpp` (pre-compiled bindings)

**Why Bun/TS?**
Node.js is too slow for CLIs, and requiring the user to install Node is poor UX. However, Bun allows compiling a TypeScript application down into a standalone binary exactly like Go. 

---

*(Note: The rest of the blueprint assumes Option A or B is chosen, establishing a single-language monorepo rather than a multi-language microservice architecture).*

## 3. Monorepo Structure

```txt
neuron-cli/
├── cmd/
│   └── neuron/                # The main entrypoint
├── internal/                  # Private application code
│   ├── tui/                   # UI views, state machines (Bubbletea or Ink components)
│   ├── cli/                   # Command parsing, flag handling
│   ├── config/                # Settings, policies, state validation
│   ├── llm/                   # Model routing, Ollama/llama.cpp adapters, Claude/OpenAI APIs
│   ├── mcp/                   # Model Context Protocol server/client orchestration
│   ├── channels/              # Slack/WhatsApp adapters
│   ├── sandbox/               # Execution environments (none, docker)
│   ├── tools/                 # Built-in file, search, shell tools
│   └── telemetry/             # OpenTelemetry + local analytics
├── pkg/                       # Public SDK code (if building extensions)
├── docs/                      # Architecture, tools, docs
├── schemas/                   # JSON schemas for settings
├── Makefile / package.json    # Build scripts
└── .goreleaser.yaml           # Release automation (Option A only)
```

## 4. Core Subsystems

### 4.1 Command Surface

Base commands:
- `neuron chat` (Default interactive session)
- `neuron run -p "..."` (Headless agent mode)
- `neuron model` (List, switch remote vs local)
- `neuron tools` (Enable, disable built-in/MCP tools)
- `neuron mcp` (Attach context servers)
- `neuron config` (View/edit settings)
- `neuron channels` (Slack/WhatsApp setup)

Slash commands within the Interactive TUI:
- `/model`, `/tools`, `/mcp`, `/clear`, `/plan`, `/run`, `/git`

### 4.2 Tooling System

Built-in tools to ship immediately (Fast, Native):
- **fs:** fast read/write/list/glob (native stdlib)
- **search:** ripgrep wrapper or native Go/Rust equivalent regex parser
- **shell:** Safe execution with TUI approval prompt
- **git:** Status, diff, commit helpers
- **web:** Fetch/Scrape using fast headless clients.

**Extensibility (MCP):**
Support the Model Context Protocol natively. The CLI acts as an MCP Client connecting to external `npx` or binary MCP servers as sub-processes. 

### 4.3 Model and Provider Layer

**The Routing Engine:**
- Normalize requests/responses (`{ role: 'user', content: '...' }`)
- Support streaming tokens to the UI natively.
- Handle JSON mode and native Tool-Calling modes.

**Local-First Architecture:**
1. **Ollama Integration:** Fast-path to `http://localhost:11434`. (Primary local provider)
2. **Cloud Fallback:** Anthropic/OpenAI/Google if `preferLocal=false` or explicitly requested. 

**Local Lifecycle Commands:**
- `neuron local pull <model>`
- `neuron local ls`
- `neuron local rm <model>`
- `neuron local start/stop <model>`

### 4.4 Policy + Sandbox

Mandatory controls before executing `shell` or `fs` tools:
- **Approval Modes:** 
  - `default` (ask before every destructive action)
  - `auto_edit` (ask only on shell exec)
  - `plan` (propose all actions first)
- **Sandbox Backends:**
  - Native Host (None)
  - Docker container wrapper 

### 4.5 Persistence

- Store session history, tool logs, and cached context in a local SQLite file (`~/.config/neuron/data.db`).
- Use JSON/TOML for human-editable `.neuron.json` settings in the workspace path.

### 4.6 Remote Channel Access (Slack + WhatsApp)

When functioning as a bot backend:
- Run via `neuron channels serve`
- Expose webhook endpoints securely. 
- Map Channel Users -> Neuron Principal IDs. 
- Map Channel Threads -> Neuron SQLite Session IDs. 

### 4.7 Testing Strategy

- **Unit tests:** Extensive stdlib testing (Go `testing` or TS `vitest`).
- **Integration Tests:** Execute the compiled binary in a temporary directory via shell scripts to validate model routing and tool execution.

## 5. Implementation Milestones

### Milestone 0: Foundation (Week 1)
- Single Binary Monorepo scaffolding
- Config loader + schema validation
- Basic `neuron --help` and routing skeleton
- Compile/Build scripts pipeline in place

### Milestone 1: Interactive CLI (Week 2)
- TUI app shell (Bubbletea or Ink)
- Streaming interface
- Model adapter abstraction (OpenAI/Anthropic stubs)

### Milestone 2: The Local Engine (Week 3)
- Ollama local integration
- Local model lifecycle (`neuron local pull/run`)
- Session persistence to SQLite

### Milestone 3: Built-in Tools & Safety (Week 4)
- Local File read/write tools
- Shell execution tool with TUI Approval Prompt
- Ripgrep integration

### Milestone 4: MCP + Extensions (Week 5)
- MCP client process spawning
- Tool discovery and parsing

### Milestone 5: Channels (Week 6)
- Slack webhooks and thread mapping
- WhatsApp payload formatting

### Milestone 6: Distribution (Week 7)
- CI Matrix (Linux/macOS/Windows)
- Automated `.exe`, `.tar.gz`, and Homebrew Tap generation via GitHub Actions (GoReleaser)
