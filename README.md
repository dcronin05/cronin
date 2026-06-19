# Cronin CLI

A cross-platform webhook streaming client for Antigravity agents.

This CLI allows you to send commands, files, and clipboard contents directly to a remote Antigravity gateway. It supports true character-by-character chunk streaming, live cognitive thought streaming, background processing, and invisible proxy heartbeats.

## Installation

Download the latest binary for your operating system from the [Releases](https://github.com/dcronin05/cronin/releases) page.

## Usage

```bash
# Configure your endpoint and token first!
export CRONIN_ENDPOINT="https://your-agent-url.com/webhook/stream"
export CRONIN_TOKEN="your-super-secret-token"

# Send a basic prompt
cronin "What is the capital of France?"

# Attach a file
cronin -f my_script.py "Refactor this code"

# Attach clipboard contents (requires wl-paste, xclip, or xsel on Linux)
cronin --clip "Explain this snippet"

# Stream internal AI thoughts and tool executions in real-time
cronin --live "Search my codebase for bugs"

# Read from piped stdin
cat error.log | cronin "Why did this crash?"
```

## Compilation

Requires Go 1.21 or later.

```bash
go build -o cronin main.go
```
