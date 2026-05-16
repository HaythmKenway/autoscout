# Autoscout - AI-Driven Recon & Vulnerability Analysis Framework

## Project Overview
Autoscout is a comprehensive security reconnaissance and vulnerability analysis platform. It integrates traditional security tools (Dalfox, SQLMap, Nuclei, etc.) with AI-driven orchestration to automate the bug hunting process. It features a Burp Suite integration for traffic ingestion and a Terminal User Interface (TUI) for real-time monitoring.

## Core Architecture
- **Language**: Primarily Go (v1.21+), with a Java-based Burp Suite extension.
- **Database**: SQLite3 (`autoscout.db`), managed via `internal/db`.
- **AI Orchestration**: Logic in `pkg/orchestrator` using backends defined in `pkg/ai` (Ollama, Gemini, Codex).
- **GUI/TUI**: Built using `charmbracelet/bubbletea` and `wish` for SSH support.
- **Tools Integration**: Wrappers for external tools located in `pkg/`.

## Key Directory Structure
- `internal/`: Private application logic.
    - `db/`: Database schema, queries, and initialization.
    - `controller/`: Core business workflows and tool execution logic.
    - `scheduler/`: Background task management.
- `pkg/`: Reusable packages and tool integrations.
    - `ai/`: AI agent interfaces and backend implementations.
    - `orchestrator/`: Decisions and routing of HTTP requests to security tools.
    - `gui/`: TUI implementation (Bubbletea models and views).
    - `burp/`: Server to receive traffic from the Burp extension.
- `extensions/burp/`: Java source code for the Burp Suite extension.

## Coding Conventions
- **Error Handling**: Use `localUtils.CheckError(err)` for critical errors or `localUtils.Logger(msg, level)` for logging (1: Info, 2: Error).
- **Concurrency**: Extensive use of goroutines for background scanning and AI analysis. Ensure thread-safety when accessing shared resources.
- **Database**: Always use the `OpenDatabase()` function from `internal/db` to get a connection. Ensure tables are initialized via `CheckTables()`.
- **AI Integration**: New AI backends must implement the `AIAgent` interface in `pkg/ai/ai.go`. Structured output is expected via the `AIPlan` struct.
- **Configuration**: Global settings are stored in `~/.config/autoscout/user-config.yaml`.

## Development Workflows

### Database Changes
1. Add new table creation logic in `internal/db/db.go`.
2. Add queries in `internal/db/queries.go` or specific domain files (e.g., `vulnerabilities.go`).
3. Update `CheckTables()` to include the new initialization logic.

### Adding a New Tool
1. Create a wrapper in `pkg/tools/` or a dedicated package if complex.
2. Register the tool in `internal/controller/`.
3. Update the `AIAction` handling in `pkg/orchestrator/orchestrator.go` to support the new tool.

### Modifying the GUI
1. Locate the relevant component in `pkg/gui/` (e.g., `dashboard.go`, `target.go`).
2. Autoscout uses the Bubbletea Elm-like architecture (Model, Update, View).
3. Ensure responsiveness for both local terminal and SSH sessions.
4. **Analysis Tab**: Supports toggling word wrap with the `w` key and clearing the feed with the `c` key.

## Testing
- Run unit tests: `go test ./...`
- Verify database migrations/initialization: `go run main.go -reset` (Warning: Clears data).
- For AI logic, use mock `AIAgent` implementations to verify `AIPlan` processing.
