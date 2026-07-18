# Spec: live-updates

## ADDED Requirements

### Requirement: Watch registered projects for file changes
The system SHALL watch each registered project's `openspec/` directory recursively for file creations, writes, renames, and deletions, and SHALL re-parse the affected project after a debounce window (~300ms) so bursts of events (editor saves, agent runs) trigger one reload.

#### Scenario: Agent checks off a task
- **WHEN** an external process edits `tasks.md` while the app is running
- **THEN** within about one second the read model reflects the new checkbox state without user action

#### Scenario: Event burst coalesced
- **WHEN** ten file events arrive within the debounce window
- **THEN** the project is re-parsed once, not ten times

### Requirement: Push updates to the browser over SSE
The system SHALL expose a Server-Sent Events endpoint that notifies connected browsers when a project's state changes, and the frontend SHALL refresh affected views upon receiving events (invalidate and refetch). The connection SHALL recover automatically after interruption.

#### Scenario: Board updates live
- **WHEN** the board is open and a watched file changes on disk
- **THEN** the affected cards update without a manual page refresh

#### Scenario: Reconnect after sleep
- **WHEN** the SSE connection drops (e.g., machine sleep) and connectivity returns
- **THEN** the frontend reconnects and resynchronizes state from the API

### Requirement: Live activity strip
The system SHALL derive human-readable activity entries by diffing consecutive project snapshots (e.g., "tasks.md · 1.3 checked", "proposal.md edited") and display the most recent entries with relative timestamps in a persistent activity strip. Entries derive only from observed file diffs; the strip MUST NOT invent actor identity.

#### Scenario: Checkbox change narrated
- **WHEN** a task changes from unchecked to checked on disk
- **THEN** the activity strip shows an entry naming the file, the task id, the transition, and a relative time

#### Scenario: Manual refresh fallback
- **WHEN** the user triggers the manual refresh action
- **THEN** all registered projects re-parse and the UI updates, regardless of watcher state
