# Design: build-storyboard-v1

## Context

Storyboard is a greenfield app in an empty repository. OpenSpec projects store all state as markdown files under `openspec/` (changes with `proposal.md`, `design.md`, `specs/**/*.md`, `tasks.md`; archive under `changes/archive/`; schema config in `config.yaml`). There is no database to integrate with — the filesystem is the system of record, and it is concurrently edited by humans and AI agents. The author is new to Go and building this as a portfolio project; the architecture is deliberately layered so the domain core is a readable, testable, UI-independent Go package.

## Goals / Non-Goals

**Goals:**

- Zero-install portability: one static binary per OS; delete the file to uninstall.
- Truthful UI: every pixel derivable from files on disk; external edits always win.
- Safe writes: checkbox toggles and text edits that never corrupt or reformat untouched markdown.
- Live board: file changes (e.g., an agent checking off tasks) visible in the UI within ~1s.
- Teachable codebase: library-first `internal/` packages, table-driven tests, idiomatic concurrency.

**Non-Goals:**

- Not a replacement for the `openspec` CLI (no scaffolding, validation, or archiving in v1).
- No creation of new changes/artifacts from the UI (view + edit existing content only).
- No auth, multi-user, cloud sync, or non-localhost network access.
- No per-task states beyond what files store (no invented "In Progress" for tasks).
- No dark theme in v1 (tokens structured so one can be added later).

## Decisions

### D1: Portable Go server + browser, not Wails/Fyne/Electron

The hard requirement is "runs everywhere, no install." Wails depends on system webviews (WebView2, webkit2gtk) that may be absent; Electron ships ~150MB and installers; Fyne is portable but makes a Jira-like board impractical. A Go `net/http` server with the UI embedded via `go:embed`, opening the default browser at `http://localhost:<port>`, is fully static, cross-compilable from one machine (`GOOS=windows|darwin|linux`), and uses the one runtime every machine has: a browser.

### D2: React + TypeScript + Vite SPA over htmx/server-rendered

htmx would maximize Go, but the author explicitly wants React on their CV (dominant share of frontend job listings). Consequence: the backend becomes a clean JSON REST API + SSE — itself a more marketable architecture. Node/npm exist only at dev time; `vite build` output (`dist/`) is embedded into the binary. Dev mode runs Vite (:5173) proxying `/api` to Go (:8080); release mode is the single binary. The Go server serves `index.html` for unknown non-`/api` paths (SPA fallback) so deep links work.

### D3: Parse `openspec/` directly; no runtime CLI dependency

Alternatives: shell out to `openspec ... --json` (robust contract, but requires Node+CLI installed, ~1s per call, and lacks per-checkbox detail) vs. parse markdown directly. Direct parsing is required anyway for checkbox granularity and keeps the binary standalone. The parser targets the observed on-disk format: `## N. Group` headings and `- [ ] N.M text` / `- [x]` lines in `tasks.md`, artifact presence for pipeline status, `config.yaml` for schema name. Format drift across OpenSpec versions is mitigated by lenient parsing (unknown lines pass through untouched) and versioned test fixtures.

### D4: Change lifecycle derived, never stored

Board columns are computed per change: **Draft** (proposal exists, tasks.md missing or empty), **In Progress** (tasks.md has ≥1 checked and ≥1 unchecked box, or any artifact beyond proposal exists), **Complete** (all checkboxes checked), **Archived** (lives under `changes/archive/`). Tasks themselves have only the two states the file can express: unchecked/checked. Storyboard never persists its own status metadata — this is the "UI never lies" invariant, and it means deleting Storyboard loses nothing.

### D5: Line-surgery writes, not parse–serialize round-trips

Rewriting markdown through a parser/serializer would normalize whitespace and mangle user formatting. Instead, all writes are line-level splices on the raw file bytes: a checkbox toggle replaces `[ ]`↔`[x]` within one line; a text edit replaces exactly the lines the user edited. Writes are atomic (write temp file in same directory, rename over original) and guarded by a staleness check: the API carries the file's modtime/hash from read time, and a mismatch rejects the write with 409 (the UI then reloads from disk — "disk always wins"). Proposal-text editing edits the raw markdown section, not a rich-text projection.

### D6: fsnotify → debounce → snapshot → SSE

A watcher goroutine per registered project watches `openspec/` recursively. Events are debounced (~300ms) on a channel with a timer, since editors and agents write in bursts. On quiet, the affected project is re-parsed into an immutable snapshot; a diff against the previous snapshot yields human-readable activity events ("tasks.md · 1.3 checked") pushed over one SSE endpoint (`/api/events`). SSE over WebSockets: one-directional needs only, auto-reconnect for free, trivial in Go, and TanStack Query invalidation on event receipt keeps the frontend logic tiny.

### D7: Registry in OS user-config dir

The multi-project registry (registered folders, recents, window prefs) is one JSON file at `os.UserConfigDir()/storyboard/config.json` — the only file Storyboard writes outside user projects. Registered paths are validated on load; missing folders are shown as disconnected, not silently dropped.

### D8: Package layout (library-first)

```
storyboard/
├── cmd/storyboard/main.go        # flag parsing, wiring, browser launch
├── internal/openspec/            # parser + domain model (zero deps on http)
├── internal/registry/            # project registry persistence
├── internal/watch/               # fsnotify + debounce + snapshot diffing
├── internal/server/              # http handlers, SSE, embed, SPA fallback
└── frontend/                     # React + TS + Vite (dist/ embedded)
```

`internal/openspec` is the showcase package: pure functions, table-driven tests against fixture directories, no I/O beyond `io/fs` (so tests can use `fstest.MapFS`).

### D9: Implementation phasing (risk last, learning first)

1. Parser package + fixtures (structs, slices, `io/fs`, table-driven tests)
2. Read-only API + single-project board UI (http, JSON, React basics)
3. Checkbox toggle (first write path: atomicity, staleness, line surgery)
4. fsnotify + SSE live updates (goroutines, channels, debounce)
5. Text editing (hardest round-tripping — after write path is proven)
6. Multi-project registry + dashboard (state fan-out — after one project works)
7. Embed, cross-compile, release polish

## Risks / Trade-offs

- [OpenSpec format drift across versions] → Lenient line-based parsing; fixtures captured from real projects; unknown content preserved verbatim; parser failures degrade to "raw view" rather than blank UI.
- [Write races with an agent editing the same file] → Atomic rename + modtime/hash precondition + 409-then-reload; the human's stale edit is rejected, never merged blindly.
- [fsnotify platform quirks (recursive watch differences, editor rename-swap saves)] → Watch directories not files; treat rename/create/write uniformly as "dirty"; debounce absorbs event storms; manual refresh button as escape hatch.
- [Browser differences / no browser auto-open on some Linux setups] → Print the URL to stdout always; auto-open is best-effort (`xdg-open`/`open`/`rundll32`).
- [Author new to Go + ambitious scope (editing, multi-project)] → Phasing in D9 ships a useful read-only board early; risky capabilities land last and are cuttable without hollowing the app.
- [Port collisions on localhost] → Bind port 0 (OS-assigned free port) by default with `--port` override.
