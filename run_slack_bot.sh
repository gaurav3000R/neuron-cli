#!/usr/bin/env bash

# Run the Neuron Slack bot using Gemini 1.5 Pro
# Make sure your GEMINI_API_KEY, SLACK_BOT_TOKEN, and SLACK_APP_TOKEN
# are set in your local .env file.

echo "Starting Neuron Slack Bot with Gemini 1.5 Pro..."
go run ./cmd/neuron slack --provider gemini --model gemini-1.5-pro
