# Neuron CLI

Neuron represents a monumental leap in personal AI assistants. It is a 100% Go-powered CLI tool that connects to local LLMs (like Ollama), providing a suite of advanced features without relying on external cloud endpoints. Your data remains perfectly safe and strictly on your machine.

## Architecture

Neuron is built as a **Single Binary Monolith**. 
1. The **Go Backend** acts as a resilient router/controller.
2. The **Tool Registry** grants the LLM safe, sandboxed access to your Local Filesystem, Shell operations, and external MCP servers.
3. The **Embedded UI** uses Golang's `//go:embed` to ship an entire React SPA within the CLI executable. 

You do not need Node.js, Python, or nginx to run this in production. One file. Everything included.

## Features

*   **Local LLM Integration**: Connects to any local Ollama instance out of the box.
*   **Built-in Tools**: `read_file`, `write_file`, and `run_command` tools allow the agent to traverse your project autonomously.
*   **Sandbox Approval**: By default, Neuron will require manual approval before executing shell commands or writing files.
*   **MCP Support**: Fully supports the standard *Model Context Protocol* via generic Stdio adapters. Bring SQLite, GitHub, or any other MCP server to your local LLM.
*   **Beautiful TUI**: An interactive Bubbletea-powered terminal UI for seamless chatting.
*   **Embedded Graphical Web UI**: A stunning dark-mode React application powered by Vite and Tailwind V4, served natively over Go endpoints via Server-Sent Events.

## Installation

### Prerequisites

*   **Go** 1.22+
*   **Node.js / npm** (Only required for compiling the React UI from source)
*   **Ollama**, configured and running locally (`ollama serve`). 

### Build From Source

```bash
# 1. Clone the repository
git clone https://github.com/your-username/neuron-cli.git
cd neuron-cli

# 2. Build the React Frontend and the Go Binary simultaneously
npm run build

# 3. The CLI is now available at ./bin/neuron
./bin/neuron help
```

*(Alternatively, wait for the GitHub Action to release pre-compiled binaries via GoReleaser!)*

## Usage Guide

Neuron commands are intuitive and fast. To get started, make sure your Ollama instance is active. By default, it looks for the `llama3` model on `http://localhost:11434`.

### Startup the Terminal UI
Launch the interactive command-line interface. 
```bash
neuron chat
```

### Startup the Web UI
Launch the fully graphical React web interface. This spins up the local Go HTTP instance on port 3133.
```bash
neuron ui
```

## Advanced Settings
You can configure the base URL or Default Model via the configuration file placed automatically in your home directory at `~/.config/neuron/config.yaml`.
