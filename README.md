# Autoscout
A comprehensive Recon & Vulnerability Analysis Framework with AI-driven orchestration.

Autoscout automates popular bug bounty hunting tools into a unified platform, featuring deep integration with Burp Suite and AI-powered reasoning backends (Ollama, Gemini, Codex).

### Core Features
- **AI Orchestrator**: Uses LLMs to analyze HTTP traffic and decide which security tools to run (Dalfox, SQLMap, Nuclei, FFUF, etc.).
- **Burp Suite Integration**: Seamlessly captures traffic from Burp and feeds it into the AI analysis pipeline.
- **Deep Traffic Analysis**: Captures full HTTP context, including Method, URL, Headers, and Body (Base64 decoded).
- **Stealth & Rate Limiting**: AI-managed rate limiting to prevent IP blocking on major platforms.
- **AI Training (Knowledge Base)**: Customize the AI's reasoning by adding your own expertise and heuristics to `~/.config/autoscout/knowledge.md`.
- **Dynamic Wordlist Resolution**: Automatically finds or generates wordlists for fuzzing tools.
- **Multi-Mode Operation**:
  - **Daemon Mode**: Continuous automated scanning.
  - **SSH Mode**: Access the GUI and control center over SSH (port 2222).
  - **GUI Mode**: Interactive Terminal UI for real-time monitoring.

### Installation
```bash
go install -v github.com/HaythmKenway/autoscout@latest
```

### Configuration
1. **Notify**: Setup `$HOME/.config/notify/provider-config.yaml`.
2. **User Config**: Global settings at `$HOME/.config/autoscout/user-config.yaml`.
3. **AI Training**: Add your custom heuristics to `$HOME/.config/autoscout/knowledge.md`.

### Project Status
- [x] Subdomain Enumeration & Tracking
- [x] Discord/Slack Notifications
- [x] Burp Suite Extension for Traffic Ingestion
- [x] AI-Powered Tool Selection & Execution
- [x] Full HTTP Context Analysis (Headers/Body)
- [x] Automated Rate Limiting & Stealth
- [x] Custom Knowledge Base for AI Training
- [ ] Validating and Processing all the URLs
- [ ] Port Scanning & Service Discovery

---
*Built for automation. Driven by AI.*
