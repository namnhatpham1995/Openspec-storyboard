# Proposal: build-storyboard-v1

## Why

OpenSpec state lives in markdown files scattered across `openspec/` directories — checking "where is this project, and what's next?" means `cat`-ing proposals and counting checkboxes by hand, and there is no way to watch an AI agent's progress in real time. Storyboard gives developers a Jira-like visual board over their OpenSpec projects: proposals read as stories, tasks as tickets, with live updates as files change on disk.

## What Changes

- New standalone application **Storyboard**: a portable, zero-install, single-binary local app (Windows/macOS/Linux) that opens a board UI in the default browser.
- Go backend that discovers and parses `openspec/` directories directly (changes, artifacts, task checkboxes, archive) — no runtime dependency on the `openspec` CLI or Node.
- Multi-project dashboard: users register several project folders and see all changes across them, grouped by lifecycle (Draft / In Progress / Complete / Archived) derived purely from files on disk.
- Change detail view: task groups with clickable `[ ]`/`[x]` checkbox glyphs, artifact pipeline status (proposal → specs → design → tasks), rendered markdown artifacts.
- Write-back with formatting preservation: toggling a checkbox or editing task/proposal text rewrites only the affected markdown lines; untouched formatting survives round-trips.
- Live updates: filesystem watching (fsnotify) pushes changes to the UI over SSE — external edits (e.g., an AI agent checking off tasks) always win and appear within moments, narrated in a live activity strip.
- React + TypeScript SPA embedded into the Go binary via `go:embed`; "drafting table" design language per project conventions.

## Capabilities

### New Capabilities

- `openspec-parsing`: Discover `openspec/` roots and parse changes, artifact files, task groups, and checkbox states into a typed read model; derive change lifecycle status from files alone.
- `project-registry`: Register, list, and remove project folders; persist the registry and recent-projects list in a local app-config file; validate registered paths on load.
- `project-dashboard`: Multi-project overview UI — lifecycle board columns with change cards showing name, phase, and task progress; project switching and empty states that teach.
- `change-detail`: Per-change view — task groups with toggleable checkbox glyphs, artifact pipeline status, rendered markdown, edit affordances for task and proposal text.
- `markdown-writes`: Safe write-back to markdown files — checkbox toggling and text edits with byte-preserving round-tripping of untouched content, atomic writes, and conflict handling (disk always wins).
- `live-updates`: Filesystem watching with debounced reload, SSE event stream to the browser, and the live activity strip narrating what changed, where, and when.
- `app-distribution`: Portable delivery — single static binary per OS with the built SPA embedded, localhost server on a free port, auto-open default browser, cross-compiled release builds.

### Modified Capabilities

<!-- none — greenfield project, no existing specs -->

## Impact

- New codebase in this repository: Go module (`storyboard`) with library-first `internal/` packages plus a `frontend/` React + TypeScript + Vite app (Node is dev-time only, never shipped).
- New dev-time dependencies: Go toolchain, Node/npm, fsnotify, chi or stdlib `net/http`, React, @dnd-kit, TanStack Query.
- Writes to user files: only the markdown files the user explicitly edits/toggles through the UI, plus one app-config file in the OS user-config directory for the project registry.
- No network access beyond localhost; no telemetry; no install/uninstall footprint beyond the binary itself.
