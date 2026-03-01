#!/bin/bash
# Direct test to see what Ollama returns

curl -X POST http://localhost:11434/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2:3b",
    "messages": [{"role": "user", "content": "hi"}],
    "stream": true,
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "read_file",
          "description": "Reads a file",
          "parameters": {
            "type": "object",
            "properties": {"path": {"type": "string"}},
            "required": ["path"]
          }
        }
      }
    ]
  }' 2>&1 | head -10
