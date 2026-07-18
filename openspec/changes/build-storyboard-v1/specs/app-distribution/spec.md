# Spec: app-distribution

## ADDED Requirements

### Requirement: Single portable binary
The system SHALL be delivered as one statically linked executable per OS/architecture (Windows, macOS, Linux) with the built frontend embedded via `go:embed`. Running it MUST NOT require installing any runtime, framework, or the OpenSpec CLI, and MUST NOT write outside the user-config directory and explicitly edited project files.

#### Scenario: Run from a USB stick
- **WHEN** the binary is copied to a machine with no Go, Node, or OpenSpec CLI installed and executed
- **THEN** the app starts and serves the full UI

#### Scenario: Uninstall by deletion
- **WHEN** the user deletes the binary and the app-config directory
- **THEN** no trace of the app remains on the system

### Requirement: Localhost server with browser launch
The system SHALL bind to a localhost port (OS-assigned free port by default, overridable via `--port`), always print the URL to stdout, and attempt to open the default browser at that URL. Failure to open a browser MUST NOT prevent the server from running. The server MUST NOT listen on non-loopback interfaces.

#### Scenario: Default launch
- **WHEN** the user runs the binary with no arguments
- **THEN** a free port is chosen, the URL is printed, and the default browser opens to the board

#### Scenario: Headless environment
- **WHEN** no browser can be opened (e.g., a Linux setup without xdg-open)
- **THEN** the server keeps running and the printed URL lets the user open the UI manually

### Requirement: SPA serving with API separation
The system SHALL serve the embedded frontend for non-`/api` paths, returning `index.html` for unknown paths so client-side routes deep-link correctly, while `/api/*` paths serve only JSON and SSE.

#### Scenario: Deep link
- **WHEN** the browser requests `/changes/build-storyboard-v1` directly
- **THEN** the server returns the SPA shell and the client router shows that change's detail view

### Requirement: Reproducible cross-platform release builds
The system SHALL provide a scripted build that cross-compiles release binaries for windows/amd64, darwin/amd64, darwin/arm64, and linux/amd64 from a single machine, embedding the current frontend build and a version string.

#### Scenario: One-command release
- **WHEN** the release script runs on the development machine
- **THEN** it produces all four binaries, each self-contained and reporting the same version via `--version`
