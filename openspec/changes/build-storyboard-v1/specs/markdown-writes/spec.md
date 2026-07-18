# Spec: markdown-writes

## ADDED Requirements

### Requirement: Formatting-preserving line surgery
The system SHALL perform all file modifications as line-level edits on the raw file bytes: a checkbox toggle replaces only `[ ]`/`[x]` within the task's line; a text edit replaces only the edited lines. All other bytes — whitespace, unknown markdown, line endings (CRLF vs LF), trailing content — MUST be preserved verbatim.

#### Scenario: Toggle preserves the rest of the file
- **WHEN** a checkbox on line 7 is toggled in a file containing prose, blank lines, and CRLF line endings
- **THEN** the resulting file differs from the original only in the `[ ]`→`[x]` characters on line 7

#### Scenario: Round-trip identity
- **WHEN** a task's text is replaced with identical text
- **THEN** the file bytes are unchanged

### Requirement: Atomic writes
The system SHALL write files atomically: content is written to a temporary file in the same directory and renamed over the original, so a crash or concurrent reader never observes a partially written file.

#### Scenario: No partial writes
- **WHEN** a write is interrupted before completion
- **THEN** the original file remains intact and unmodified

### Requirement: Staleness detection — disk always wins
The system SHALL require each write request to carry the file version (modtime and/or content hash) it was based on, and SHALL reject the write with a conflict response when the file has since changed on disk. On conflict the UI MUST reload from disk and MUST NOT retry automatically or attempt a merge.

#### Scenario: Agent edited the file first
- **WHEN** an AI agent modifies `tasks.md` after the UI read it, and the user then saves an edit based on the old version
- **THEN** the write is rejected with a conflict, the UI reloads the agent's version, and the user's edit is not blindly applied

#### Scenario: Fresh write succeeds
- **WHEN** the file is unchanged since the UI read it and the user saves
- **THEN** the write applies and the response carries the new file version
