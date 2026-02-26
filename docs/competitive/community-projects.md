# Community Projects

Smaller projects shared on r/ClaudeAI and similar forums. These range from polished tools to weekend hacks, showing the breadth of approaches in this space.

## Notable Projects

### AgentOS
**GitHub:** https://github.com/saadnvd1/agent-os
**Status:** Active (208 commits, v0.1.0 Jan 2026)

Mobile-first web interface with strong feature set:
- Multi-agent: Claude, Codex, OpenCode, Gemini, Aider, Cursor
- Git worktree support for isolated branches
- Multi-pane terminal (up to 4 sessions)
- Code search with syntax highlighting
- Git integration (status, diffs, commits, PRs)
- MCP orchestration
- Desktop apps via Tauri

Tech: TypeScript, Next.js, Tauri

**Note:** More polished than typical community project. Has commercial cloud version.

---

### Termly CLI
**GitHub:** https://github.com/termly-dev/termly-cli
**Status:** Active (v1.7)

Universal PTY wrapper for any terminal AI tool:
- Works with 20+ AI assistants (not just Claude)
- E2E encryption (AES-256-GCM, Diffie-Hellman)
- Session management with auto-reconnect
- Prebuilt binaries for all platforms
- 100KB circular buffer for session resumption

Tech: Node.js, WebSocket, node-pty

---

### Obsidian Claude Anywhere
**GitHub:** https://github.com/derek-larson14/obsidian-claude-anywhere
**Status:** Active (v1.0.9, 36 commits)

Obsidian plugin with embedded relay:
- Full terminal access via xterm.js
- Skill commands (/morning, /mail, /voice)
- Session resumption via /resume
- File editing across Mac filesystem
- Tailscale networking

Tech: JavaScript, Python relay (embedded), xterm.js

**Note:** Has the terminal page feature HAPI has.

---

### Geoff
**GitHub:** https://github.com/belgradGoat/Geoff
**Status:** Early (18 commits, Jan 2026)

"Side projects while you're at work" framing:
- Multi-agent: Claude Code, OpenAI Codex
- Supabase for task sync
- Tailscale networking
- File browsing from phone

Tech: Python backend, TypeScript/React frontend, Supabase

---

### VSClaude WebApp
**GitHub:** https://github.com/khyun1109/vscode_claude_webapp
**Status:** Early (2 commits)

Scrapes VS Code via Chrome DevTools Protocol:
- Real-time monitoring via HTML snapshots
- Multi-tab session support
- Push notifications
- Auto-edit mode toggle

Tech: Node.js, Express, WebSocket, Chrome DevTools Protocol

**Note:** Fragile approach but shows demand for the use case.

---

### Moshi
**Website:** https://getmoshi.app/
**Status:** Beta (free during beta, iOS App Store)

Mobile terminal specifically for AI coding agents:
- Uses Mosh protocol for network resilience (survives connection switches, device sleep)
- Voice-to-terminal via on-device dictation
- Face ID security for SSH keys in Keychain
- Native tmux integration
- Task completion notifications
- Custom terminal keyboard (Ctrl, Esc, arrows)
- Session recovery if interrupted by iOS
- Multiple themes (Nord, Dracula, Solarized)

Platform: iOS (Android in development per GitHub)

Tech: Native iOS, Mosh protocol

**Note:** Different approach than web-based solutions. Native app with Mosh gives better connection resilience than WebSocket-based approaches. Marketing specifically targets "Check on Claude Code from the couch" use case.

---

### Codex Pocket
**GitHub:** https://github.com/ddevalco/codex-pocket
**Status:** Active (278 commits, MIT)

iPhone remote control for Codex Desktop via Tailscale:
- Tailscale-only networking (no public exposure)
- Device pairing via QR codes with per-device session tokens
- SQLite event persistence and replay
- Image attachments with local storage
- Thread title sync with Codex Desktop state
- Per-thread progress tracking, concurrent composition
- Read-only GitHub Copilot CLI session viewing (ACP protocol)
- launchd agent for macOS background service
- Markdown/JSON thread export with iOS share sheet

Tech: Svelte, TypeScript, Bun, SQLite, Vite

**Note:** Fork of "Zane" project, stripped down to focus on local Codex control. iPhone-first UX (Enter = newline, Cmd+Enter = submit). Codex-only write support.

---

### claude-devtools
**GitHub:** https://github.com/matt1398/claude-devtools
**Website:** claude-dev.tools
**Status:** Active (120 commits, v0.1.0, MIT, Feb 2026)

Read-only desktop app that visualizes Claude Code sessions from raw JSONL logs:
- Per-turn context attribution across 7 token categories (CLAUDE.md, tool outputs, thinking, etc.)
- Compaction phase tracking (when/how context was compressed)
- Rich tool viewers: syntax-highlighted code, inline diffs, search results
- Subagent execution trees (expandable Task→subagent→sub-subagent chains)
- Team coordination visualization (color-coded teammate cards)
- Custom notification triggers with regex matching and ReDoS protection
- SSH remote session viewing (SFTP to remote `~/.claude/`)
- Multi-pane drag-and-drop layout for side-by-side session comparison
- Cross-session search (Cmd+K command palette)
- Docker standalone mode (read-only mount, zero telemetry)

Tech: Electron, React, Zustand, Fastify, Tailwind, TypeScript, pnpm

**Note:** Complementary to supervisor tools like yepanywhere — this is post-hoc analysis ("what happened in this session") vs real-time control ("drive and approve sessions"). Reads the same `~/.claude/` files. Strong context attribution UI worth studying. See [detailed architecture analysis](../research/claude-devtools.md).

---

### Farfield
**GitHub:** https://github.com/achimala/farfield
**Status:** Active (84 commits, 19 stars, MIT)

Local web UI for remote-controlling Codex Desktop:
- Thread browser organized by project
- Chat interface with model and reasoning effort controls
- Plan mode toggle per thread
- Real-time agent event streaming via SSE
- Interrupt controls
- Debug tab with full IPC history, payload inspection, and replay
- Zod schemas as single source of truth (hard fail on unknown payloads)

Tech: React, TypeScript, Vite, Tailwind, pnpm monorepo

**Note:** Connects directly to Codex Desktop's IPC socket and HTTP app-server API. Debug/IPC inspection tab is unique among competitors. Codex-only.

---

### Cogpit
**GitHub:** https://github.com/gentritbiba/cogpit
**Website:** cogpit.gentrit.dev
**Status:** Active (v0.0.7, MIT)

Polished desktop + web dashboard for browsing, inspecting, and interacting with Claude Code sessions (Claude-only):
- **Electron + Web** from single codebase (electron-vite), with macOS/Linux installers
- Live streaming via SSE with file watching on `~/.claude/projects/` JSONL files
- Subagent synthesis — watches child agent JSONL files and merges progress into parent timeline
- Undo/redo with full branching — archives turns, reverses file operations, branch switcher modal
- Token analytics with per-turn cost breakdown (cache read/creation, model-aware pricing, SVG charts)
- Team dashboards — kanban task board, member status grid, inter-agent message timeline
- Voice input via Whisper WASM (client-side, no external API)
- Persistent Claude processes (keeps prompt cache warm via `--resume`)
- Virtualized turn list (@tanstack/react-virtual) for large sessions
- Optional network access with password auth, local clients bypass auth
- Responsive layout: 3-column desktop, tab-based mobile

Tech: React 19, Vite 6, Electron, Express 5, Tailwind 4, Radix UI, Shiki, node-pty, Bun

**Note:** CLI-based integration (spawns `claude` with `--output-format stream-json`), not SDK-native. Strongest on analytics and session inspection — the token/cost charts and branching undo are features we don't have. Single author (Gentrit Biba). 870 unit tests (Vitest). No relay or E2E encryption — remote access is basic password + bind to `0.0.0.0`. Tightly coupled to Claude Code JSONL format.

---

### Claude Code UI (CloudCLI)
**GitHub:** https://github.com/siteboon/claudecodeui
**Website:** cloudcli.ai
**Status:** Active (v1.18.2, GPL-3.0)

Web UI for multiple AI coding agents with built-in terminal and file editor:
- Multi-agent: Claude Code (SDK-native), Cursor CLI, OpenAI Codex
- Built-in terminal (xterm.js) and file explorer with CodeMirror editor
- Git explorer (status, diffs, commits, branch switching)
- Granular tool permission controls (strict/default/permissive modes)
- MCP server management through UI
- TaskMaster AI integration for project planning
- i18n support (English, Japanese, Korean, Chinese)
- PWA with mobile-optimized layout
- Single-user JWT auth with SQLite + bcrypt

Tech: React, Express, Vite, SQLite (better-sqlite3), node-pty, TypeScript/JSX, Tailwind

**Note:** One of few projects using the Claude Agent SDK directly (v0.1.71) rather than wrapping the CLI. More of a local dev companion than a remote supervisor — no relay, no E2E encryption, no push notifications. Large monolithic server files (index.js ~1930 lines, projects.js ~1826 lines). No visible tests. `npx @siteboon/claude-code-ui` one-liner install is a nice touch.

---

### Codex Web UI
**GitHub:** https://github.com/friuns2/codex-web-ui
**Status:** Active (29 commits, 46 stars, v0.1.0, MIT)

Runtime-patching toolkit that cracks open Codex Desktop's Electron app to expose full UI over HTTP/WebSocket:
- Extracts `app.asar`, injects runtime HTTP+WS server patch, runs Electron headless
- Any browser on the network gets the full Codex UI — no client install
- Hand-rolled RFC 6455 WebSocket (zero npm deps in the injected code)
- Token-based auth (timing-safe), rate limiting, single-active-client policy
- Unlocks Codex Desktop's **hidden SSH remote execution engine** — fully compiled in the binary but never wired up in the UI
- SSH mode: auto-selects remote host, key-based auth, remote git apply, command execution
- Minification-safe patching strategies that survive Codex bundle updates
- `npx -y codex-web-ui --port 5999` one-liner

Tech: Bash (launcher), vanilla JS (bridge ~200 lines, server ~900 lines injected)

**Note:** Fundamentally different approach — not a wrapper or SDK integration. It's reverse engineering of the proprietary Electron app. Fragile (breaks when Codex updates bundle shape) but proves the Electron window is a transport limitation, not a technical one. The SSH unlock is the most interesting finding — OpenAI shipped remote execution but never exposed it. No session management, no multi-agent, no relay — it's a raw transport enabler, not a supervisor.

---

### Chroxy
**GitHub:** https://github.com/blamechris/chroxy
**Status:** Active (700+ commits, v0.1.0, MIT)

React Native mobile app + Node.js daemon for remote Claude Code supervision:
- Dual-mode: clean chat UI **and** full xterm.js terminal (PTY/tmux mode)
- Cloudflare Quick Tunnels (zero-config, no account) or Named Tunnels (stable URL)
- LAN discovery via mDNS/Bonjour
- E2E encryption (NaCl X25519 + XSalsa20-Poly1305 — same scheme as yepanywhere)
- Voice input via `expo-speech-recognition`
- QR code connection setup
- Plan mode approval UI (inline card in chat view)
- File browser with syntax highlighting + diff viewer
- Push notifications via Expo Push Service
- Provider registry: pluggable backends (SDK, legacy CLI, PTY) with capability flags
- Claude Agent SDK v0.1.0 integration (in-process permissions, live model switching)
- Session ring buffer (500 messages in-memory, lost on crash)
- Docker support (headless CLI mode only)

Tech: Plain JavaScript server (no TypeScript, no build step), React Native (Expo SDK 54, TypeScript), Zustand, node-pty, Cloudflare tunnels

**Note:** Closest direct competitor to yepanywhere — same problem space, same E2E encryption approach. Key differences: native mobile app (no web client), Cloudflare tunnels (vs self-hosted relay), plain JS server (no type safety). High velocity but no CI/CD, no app tests (700+ server tests only), tokens stored in plaintext. The no-TypeScript server is a ceiling — 11k LOC with no compiler catching refactoring breakages. Expo app requires custom dev builds (not on app stores). Cloudflare Quick Tunnels have no SLA.

---

### claude-remote-approver
**GitHub:** https://github.com/yuuichieguchi/claude-remote-approver
**Status:** Early (minimal)

Remote approval of Claude Code permission prompts via phone notifications:
- Sends tool permission prompts (Bash, Write, Edit) as push notifications via ntfy.sh
- Tap-to-approve from phone
- 120-second timeout with auto-deny
- Configurable ntfy server (self-hostable)
- Secure topic generation (128-bit random)

Tech: JavaScript, Node.js, ntfy.sh

**Note:** Single-purpose tool — only handles approval, not session viewing or chat. Lightweight alternative to a full supervisor for users who just want mobile approvals.

---

### claude-link
**GitHub:** https://github.com/Qsanti/claude-link
**Status:** New (0 stars, MIT)

Telegram bridge for Claude Code — forwards messages between Telegram and the local CLI:
- Send prompts from phone via Telegram, get Claude Code responses back
- Voice message transcription (optional, requires OpenAI API key)
- Commands: `/cancel`, `/new`, `/status`, `/help`
- Uses existing Claude Code config, skills, and CLAUDE.md
- Single-user security with restricted file permissions
- `pip install claude-link` one-liner

Tech: Python 3.10+, Telegram Bot API, PyPI

**Note:** Minimal approach — no web UI, no session management, just a Telegram↔CLI pipe. Similar to touchgrass.sh and Claude-Code-Remote in the "chat app as remote control" category. No E2E encryption beyond Telegram's own transport.

---

### Poirot
**GitHub:** https://github.com/LeonardoCardoso/Poirot
**Status:** Active (55 stars, MIT)

Native macOS companion app for Claude Code:
- Browse and search Claude Code sessions from `~/.claude/projects/`
- Explore diffs with syntax highlighting (HighlightSwift)
- Re-run commands from session history
- Rich markdown rendering (MarkdownUI)
- Pure SwiftUI — no Electron, no web stack

Tech: Swift, SwiftUI, macOS native

**Note:** Read-only session viewer like claude-devtools but native macOS instead of Electron. Lightweight and focused — no session driving, no remote access. Targets users who want a fast, native way to browse their Claude Code history on Mac.

---

## Patterns Observed

1. **Tailscale is popular** — Many use it for secure remote access
2. **PTY/terminal streaming** — Common approach vs SDK integration
3. **E2E encryption** — Becoming expected for remote access
4. **Multi-agent support** — Growing expectation beyond Claude-only
5. **Codex-specific tools emerging** — Codex Pocket and Farfield target Codex Desktop directly via IPC/app-server APIs rather than wrapping a CLI
6. **Post-hoc analysis tools** — claude-devtools reads raw JSONL logs rather than wrapping or driving sessions, showing demand for session debugging/inspection
7. **SDK-native integration emerging** — Claude Code UI uses the Agent SDK directly rather than spawning CLI processes, suggesting the SDK is becoming stable enough for third-party use
8. **Electron reverse engineering** — codex-web-ui patches the proprietary Codex app at runtime rather than using any official API, and discovered that OpenAI shipped a fully-compiled SSH remote execution engine that was never wired up

## Last Updated

2026-02-22
