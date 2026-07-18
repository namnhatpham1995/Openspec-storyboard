# Tasks: build-storyboard-v1

Each numbered group below is **one PR** on its own branch. Workflow per group: `git checkout main` → `git pull` → `git checkout -b <branch>` → complete the group's tasks → open a PR → merge before starting the next group. Groups are ordered by dependency; a group is only started after the previous PR is merged.

## 1. PR #1 — Project scaffold (branch: `feat/scaffold`)

- [x] 1.1 Install Go toolchain; run `go version`; init module `go mod init storyboard` with `cmd/storyboard/main.go` printing the version (learn: modules, packages, `go run`)
- [x] 1.2 Create the package skeleton from design D8 (`internal/openspec`, `internal/registry`, `internal/watch`, `internal/server`) with doc comments; add `.gitignore`
- [x] 1.3 Set up `gofmt`/`go vet` habits and a `Makefile` (or `justfile`) with `run`, `test`, `build` targets

## 2. PR #2 — Parser package (branch: `feat/parser`)

- [ ] 2.1 Define the domain structs: `Project`, `Change`, `Artifact`, `TaskGroup`, `Task` with JSON tags (learn: structs, slices, exported fields)
- [ ] 2.2 Build test fixtures: copy this repo's real `openspec/` tree plus hand-made edge cases (empty change, archive, malformed tasks.md, CRLF file) into `internal/openspec/testdata/`
- [ ] 2.3 Implement `Discover(fsys fs.FS)` finding `openspec/`, config.yaml schema name, active + archived change dirs; table-driven tests with `fstest.MapFS` (learn: interfaces via `io/fs`, error wrapping)
- [ ] 2.4 Implement tasks.md parsing: `## ` groups, `- [ ]`/`- [x]` lines with id/text/state/line-number capture; preserve unknown lines verbatim in the model; table-driven tests covering every spec scenario
- [ ] 2.5 Implement artifact presence detection (proposal/specs/design/tasks) per change
- [ ] 2.6 Implement lifecycle derivation (Draft/In Progress/Complete/Archived) exactly per spec `openspec-parsing`; tests for each boundary case
- [ ] 2.7 Run `go test ./... -cover`; reach solid coverage on the parser and fix anything the fixtures expose

## 3. PR #3 — Read-only API server (branch: `feat/api-readonly`)

- [ ] 3.1 Stand up `net/http` server with a health endpoint and JSON helper (learn: handlers, `http.ServeMux`, `encoding/json`)
- [ ] 3.2 Add `GET /api/projects/current` returning the parsed model for one hardcoded project path (single-project stepping stone)
- [ ] 3.3 Add `GET /api/changes/{name}` returning full change detail including raw artifact markdown and per-file version (modtime + hash) for later staleness checks
- [ ] 3.4 Add structured logging via `log/slog` and graceful shutdown on Ctrl+C (learn: contexts, signals)

## 4. PR #4 — Read-only board UI (branch: `feat/board-ui`)

- [ ] 4.1 Scaffold `frontend/` with Vite + React + TypeScript; set up the Vite dev proxy for `/api`; add TanStack Query
- [ ] 4.2 Implement the design tokens: drafting-paper palette CSS variables, bundled IBM Plex Sans/Mono woff2, base layout shell
- [ ] 4.3 Build the lifecycle board view: columns Draft/In Progress/Complete/Archived with change cards (name, artifact pipeline, mono `n/m` progress)
- [ ] 4.4 Build the change detail view: task groups with read-only `[ ]`/`[x]` glyphs, artifact pipeline, rendered markdown viewer
- [ ] 4.5 Add keyboard navigation (tab/arrows across cards and tasks, Enter to open) and visible focus rings; empty states per spec `project-dashboard`

## 5. PR #5 — Checkbox toggle, first write path (branch: `feat/task-toggle`)

- [ ] 5.1 Implement line-surgery toggle in `internal/openspec`: flip `[ ]`↔`[x]` on one line of raw bytes, preserving CRLF/LF and all other bytes; exhaustive round-trip tests (spec `markdown-writes`)
- [ ] 5.2 Implement atomic write (temp file + rename) and version precondition check returning a typed conflict error
- [ ] 5.3 Add `POST /api/changes/{name}/tasks/{id}/toggle` carrying the base version; 409 on staleness
- [ ] 5.4 Wire the UI: click/Space toggles with optimistic-free flow (update only after success); on 409 reload from disk and notify "file changed externally"

## 6. PR #6 — Live updates (branch: `feat/live-updates`)

- [ ] 6.1 Add fsnotify recursive watching of the project's `openspec/` (watch dirs, re-add on create) in `internal/watch` (learn: goroutines, channels)
- [ ] 6.2 Implement debounce (~300ms timer reset on each event) producing "project dirty" notifications; unit-test with synthetic event streams
- [ ] 6.3 Implement snapshot re-parse + diff producing activity entries ("tasks.md · 1.3 checked"); tests for checkbox and artifact-presence diffs
- [ ] 6.4 Add `GET /api/events` SSE endpoint broadcasting change + activity events to all connected clients (learn: streaming responses, client bookkeeping with mutex or channels)
- [ ] 6.5 Frontend: subscribe to SSE, invalidate TanStack Query caches on events, auto-reconnect; verify an external edit updates the open board within ~1s
- [ ] 6.6 Build the live activity strip UI with relative timestamps and a manual refresh action

## 7. PR #7 — Text editing (branch: `feat/text-editing`)

- [ ] 7.1 Extend line surgery to replace a task's description text while preserving checkbox state, id prefix, indentation, and all other lines; round-trip identity tests
- [ ] 7.2 Add proposal editing: serve raw proposal.md, accept full-text save with version precondition (same atomic + 409 machinery)
- [ ] 7.3 API endpoints: `PUT /api/changes/{name}/tasks/{id}/text` and `PUT /api/changes/{name}/artifacts/proposal`
- [ ] 7.4 UI: inline task-text editing (edit-in-place, Esc cancels, Enter saves) and a plain-text proposal editor with save/cancel; conflict flow reloads from disk

## 8. PR #8 — Multi-project registry & dashboard (branch: `feat/multi-project`)

- [ ] 8.1 Implement `internal/registry`: load/save JSON at `os.UserConfigDir()/storyboard/config.json`, path validation, corrupt-file backup+reset (spec `project-registry`)
- [ ] 8.2 API: `GET/POST/DELETE /api/projects`; replace the hardcoded path from 3.2; one watcher per registered project
- [ ] 8.3 UI: project registration flow (folder path input with validation feedback), project list with disconnected-state handling, remove action
- [ ] 8.4 Extend the board to aggregate all projects with per-project attribution and a single-project filter
- [ ] 8.5 First-launch onboarding empty state per spec

## 9. PR #9 — Distribution & release (branch: `feat/distribution`)

- [ ] 9.1 Embed `frontend/dist` via `go:embed` with SPA fallback (index.html for unknown non-/api paths); verify deep links in the built binary
- [ ] 9.2 Bind port 0 by default with `--port` override; always print URL; best-effort browser open per OS (`rundll32`/`open`/`xdg-open`)
- [ ] 9.3 Write the release script cross-compiling windows/amd64, darwin/amd64, darwin/arm64, linux/amd64 with version stamping via `-ldflags`; verify `--version`
- [ ] 9.4 Manual cross-platform pass: run the Windows binary from a non-dev folder against a real OpenSpec project; check reduced-motion and narrow-window behavior
- [ ] 9.5 Write README with screenshots, architecture sketch, and CV-ready project description; tag v1.0.0
