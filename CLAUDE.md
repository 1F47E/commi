# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

COMMI is an AI-powered Git commit message generator written in Go. It analyzes Git status and diffs to automatically generate descriptive commit messages using LLM providers (Anthropic Claude or OpenAI).

## Common Development Commands

### Build and Test
```bash
make verify      # Run both lint and test
make test        # Run Go tests  
make lint        # Run go mod tidy and golangci-lint
make run         # Run with DEBUG=1
```

### Running the Application
```bash
go run main.go                    # Generate commit message
go run main.go -f                 # Auto-commit (YOLO mode)
go run main.go -s "feat: "        # With custom prefix
go run main.go -m "custom msg"    # With custom subject
```

## Architecture Overview

The codebase follows a clean architecture pattern with interface-based design:

- **main.go**: CLI entry point using Cobra framework
- **internal/clients/**: LLM provider implementations (Anthropic/OpenAI) with shared HTTP client
- **internal/core/**: Business logic including prompt generation and XML response parsing
- **internal/git/**: Git operations wrapper
- **internal/tui/**: Terminal UI using Bubble Tea framework
- **internal/utils/**: Utility functions

## Key Implementation Details

### LLM Provider Selection
The application automatically selects an LLM provider based on available API keys:
1. Checks `COMMI_LLM_PROVIDER` environment variable
2. Falls back to first available provider (ANTHROPIC_API_KEY or OPENAI_API_KEY)
3. Provider interface defined in `internal/core/core.go`

### XML Communication Format
The application uses XML for structured communication with LLMs:
- System prompt: `internal/core/prompt.go`
- Response parsing: `internal/core/parser.go`
- Format: `<reply><commits><commit>...</commit></commits></reply>`

### TUI Implementation
The terminal UI (`internal/tui/tui.go`) provides:
- Interactive commit message review
- Options: Commit, Copy to clipboard, Regenerate, Cancel
- Loading spinner during generation
- Styled output using lipgloss

## Environment Variables

- `ANTHROPIC_API_KEY`: Anthropic API key
- `OPENAI_API_KEY`: OpenAI API key  
- `COMMI_LLM_PROVIDER`: Force specific provider (ANTHROPIC/OPENAI)
- `DEBUG`: Enable debug logging
- `DISABLE_EMOJI`: Disable emoji in commit messages

## Testing Guidelines

When adding new features:
1. Add tests in the same package (e.g., `core_test.go` for `core.go`)
2. Run `make test` to ensure tests pass
3. Run `make lint` to check code style
4. Use `make verify` before committing