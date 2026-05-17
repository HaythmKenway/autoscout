# Autoscout - AI-Driven Recon & Vulnerability Analysis Framework

## Project Overview
Autoscout is a comprehensive security reconnaissance and vulnerability analysis platform. It integrates traditional security tools (Dalfox, SQLMap, Nuclei, etc.) with AI-driven orchestration to automate the bug hunting process. It features a high-density, professional Terminal User Interface (TUI) and deep integration with Burp Suite.

## Core Architecture
- **Language**: Primarily Go (v1.21+), with a Java-based Burp Suite extension.
- **Database**: SQLite3 (`autoscout.db`), managed via `internal/db`.
- **AI Orchestration**: Logic in `pkg/orchestrator` using backends in `pkg/ai` (Ollama, Gemini, Codex).
- **GUI/TUI**: Built with `charmbracelet/bubbletea`, `lipgloss`, and `bubblezone`.
- **Storage**: Manual investigation forensic files are stored in `/tmp/autoscout/`.

## Coding Conventions
- **Error Handling**: Use `localUtils.CheckError(err)` for critical failures; `localUtils.Logger(msg, level)` for logging.
- **Concurrency**: Use goroutines for tool execution; ensure thread-safe DB access via `OpenDatabase()`.
- **Data Integrity**: 
    - Separate encoding: Raw Burp requests/responses must be base64-encoded independently.
    - Contextual link: Every URL in the `urls` table should be linked to a `session_id` if captured from Burp.
- **AI Prompts**: 
    - Prioritize `UserContext` instructions.
    - Use `TruncateBody` to limit bodies to 10KB while keeping full headers.

## specialized UI/UX Mandates
The project follows strict professional TUI standards:
- **No Double-Delegation**: `tea.WindowSizeMsg` must be processed ONLY by the parent model to calculate sub-pane dimensions. Never delegate the raw `WindowSizeMsg` to sub-models; instead, propagate the correctly calculated `subMsg`.
- **Responsive Design**: All layout math must use `TerminalHeight - 4` as the content budget to account for global status bars and borders.
- **Frame Accounting**: Subtract border/padding overhead (usually 4 chars width, 2 lines height) from all inner component calculations.
- **Keyboard-First**: Implement vim-style `j/k` navigation, `Tab` focus cycling, and a global `?` help overlay.
- **Mouse-Augmented**: Use `bubblezone` to ensure all buttons, tabs, and table rows are clickable.
- **Visual Feedback**: The active pane MUST have a distinct `m.theme.Accent` border.

## Key Specialized Agent Skills
Always adhere to these expert guides when modifying the UI:
- **responsive-tui**: Rules for percentage-based layouts and adaptive scaling.
- **tui-ux-pro**: Professional patterns (help modals, breadcrumbs, focus states) inspired by `lazygit` and `k9s`.

## Testing & Validation
- **Unit Tests**: `go test ./...`
- **View Tests**: Use `pkg/gui/target_test.go` as a template for rendering diagnostic frames to verify UI stability across standard (120x30) and small (80x24) terminals.
- **Build**: ALWAYS run `./build.sh` after changes to verify both Go and Java components.
