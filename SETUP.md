# Setup Instructions for Neuron CLI

Depending on the architectural path you choose from `BLUEPRINT.md` (Option A: Go or Option B: Bun/TypeScript), you will need to install different toolchains to begin development. 

Here is what you need to install for your local development environment.

---

## Option A: The "Ollama/GitHub" Architecture (100% Go)

If you are proceeding with the highly recommended **Go** architecture for ultimate speed and native binaries, install the following:

### 1. The Go Toolchain (v1.22+)
You need the Go compiler to build and run the code.
- **Linux (Ubuntu/Debian):**
  ```bash
  sudo apt update
  sudo apt install golang-go
  ```
- **macOS (Homebrew):**
  ```bash
  brew install go
  ```
- **Windows:** Download the installer from [golang.org/dl](https://go.dev/dl/).

*Verify installation:* `go version` (Should be 1.22 or higher).

### 2. A C Compiler (GCC / Clang)
Because you will likely use SQLite (`go-sqlite3`) or local LLM bindings (`go-llama.cpp`) which rely on CGO (C implementations), you need a C compiler.
- **Linux:** `sudo apt install build-essential`
- **macOS:** Install Xcode Command Line Tools (`xcode-select --install`).
- **Windows:** Install GCC via MinGW or MSYS2.

### 3. Make (Optional but Recommended)
For running build scripts, installing standard `make` is very helpful.
- **Linux:** `sudo apt install make`
- **macOS:** Included in Xcode Command Line Tools.

### 4. GoReleaser (For Building/Publishing)
This tool will automate compiling your Go code into `.exe`, Linux binaries, and macOS binaries simultaneously.
- **Linux:** 
  ```bash
  go install github.com/goreleaser/goreleaser/v2@latest
  ```
- **macOS:** `brew install goreleaser`

---

## Option B: The "Modern Web" Architecture (TypeScript + Bun)

If you are proceeding with the TypeScript and Bun architecture to leverage the NPM ecosystem while still compiling to a single binary:

### 1. Bun Runtime
Bun is an all-in-one toolkit: it runs TypeScript natively, serves as a fast package manager (replacing npm/pnpm), and provides the bundler to compile your TS into a single binary.
- **Linux & macOS:**
  ```bash
  curl -fsSL https://bun.sh/install | bash
  ```
- **Windows:**
  ```powershell
  powershell -c "irm bun.sh/install.ps1 | iex"
  ```

*Verify installation:* `bun --version`

### 2. Node.js (Fallback / Tooling)
While Bun is dropping Node.js requirements, having Node.js installed is still strongly recommended for VSCode extensions, ESLint, and specific NPM edge cases.
- Use **NVM (Node Version Manager)** to install Node 20+:
  ```bash
  curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
  nvm install 22
  nvm use 22
  ```

### 3. Build Essentials (Similar to Go)
If you use `node-llama-cpp` or native SQLite bindings via Bun, you still need python and a C++ compiler to build those NPM packages locally.
- **Linux:** `sudo apt install build-essential python3`
- **macOS:** `xcode-select --install`

---

## General Tools (Regardless of Choice)

1. **Git:** For version control.
2. **VS Code (or Neovim/Zed):** 
   - *If using Go:* Install the official `Go` extension by the Go Team.
   - *If using Bun/TS:* Install `ESLint`, `Prettier`, and the official `Bun` extension.
3. **Ollama:** To test local LLM routing locally without spending API credits.
   - Install from [ollama.com/download](https://ollama.com/download)
   - Run a test model: `ollama run llama3`
