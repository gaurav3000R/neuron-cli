# Setup Instructions for Neuron CLI + Web UI

Depending on the architectural path you choose from `BLUEPRINT.md` (Option A: Go or Option B: Bun/TypeScript), you will need to install different toolchains. Because both options now include an embedded Web UI (React/Vite), you will need some frontend tooling regardless of your backend choice.

---

## 🌎 Frontend Requirements (For Both Options)

Even if you write the core CLI backend in Go, the SPA Web UI (`/ui` folder) is a Node.js project. You will need:

1. **Node.js**: (Version 20+ required). Use **NVM (Node Version Manager)** to install:
   ```bash
   curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
   nvm install 22
   nvm use 22
   ```
2. **pnpm** or **npm** (pnpm recommended):
   ```bash
   npm install -g pnpm
   ```

*(In Option A, the compiled `dist/` JS bundle will later be embedded into the Go binary at compile time via `go:embed`. In Option B, Bun handles it.)*

---

## Option A: The "Ollama/GitHub" Architecture (100% Go Core)

If you are proceeding with the highly recommended **Go** architecture for the CLI engine & backend:

### 1. The Go Toolchain (v1.22+)
You need the Go compiler to build the CLI and the embedded web server.
- **Linux (Ubuntu/Debian):**
  ```bash
  sudo apt update
  sudo apt install golang-go
  ```
- **macOS (Homebrew):**
  ```bash
  brew install go
  ```

### 2. A C Compiler (GCC / Clang)
Because you will likely use SQLite (`go-sqlite3`) or local LLM bindings (`go-llama.cpp`) which rely on CGO.
- **Linux:** `sudo apt install build-essential`
- **macOS:** Install Xcode Command Line Tools (`xcode-select --install`).

### 3. Make & GoReleaser (For Building/Publishing)
- **Linux:** 
  ```bash
  sudo apt install make
  go install github.com/goreleaser/goreleaser/v2@latest
  ```
- **macOS:** `brew install make goreleaser`

---

## Option B: The "Modern Web" Architecture (TypeScript + Bun)

If you are proceeding with the TypeScript and Bun architecture to leverage the NPM ecosystem everywhere:

### 1. Bun Runtime
Bun is an all-in-one toolkit: it runs TypeScript natively, serves as the Web API, and compiles your core + UI into a single binary.
- **Linux & macOS:**
  ```bash
  curl -fsSL https://bun.sh/install | bash
  ```

### 2. Build Essentials (Similar to Go)
If you use `node-llama-cpp` or native SQLite bindings via Bun, you still need python and a C++ compiler to build those NPM packages locally.
- **Linux:** `sudo apt install build-essential python3`
- **macOS:** `xcode-select --install`

---

## General Tools (Regardless of Choice)

1. **Git:** For version control.
2. **VS Code:** 
   - *If using Go:* Install the official `Go` extension by the Go Team.
   - *For the UI:* Install `ESLint`, `Prettier`, and Tailwind IntelliSense.
3. **Ollama:** To test local LLM routing locally without spending API credits.
   - Install from [ollama.com/download](https://ollama.com/download)
   - Run a test model: `ollama run llama3`
