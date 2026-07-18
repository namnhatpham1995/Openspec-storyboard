# Spec: openspec-parsing

## ADDED Requirements

### Requirement: Discover OpenSpec structure in a project folder
The system SHALL, given a project folder path, locate the `openspec/` directory and enumerate active changes under `openspec/changes/` and archived changes under `openspec/changes/archive/`, reading the schema name from `openspec/config.yaml` when present.

#### Scenario: Valid OpenSpec project
- **WHEN** a folder containing `openspec/changes/` with two change directories is parsed
- **THEN** the read model lists both changes with their names taken from the directory names

#### Scenario: Folder without OpenSpec
- **WHEN** a folder with no `openspec/` directory is parsed
- **THEN** parsing returns a typed "not an OpenSpec project" error and no partial model

#### Scenario: Archived changes are separated
- **WHEN** a change directory exists under `openspec/changes/archive/`
- **THEN** it appears in the read model flagged as archived, not among active changes

### Requirement: Parse tasks.md into groups and tasks
The system SHALL parse `tasks.md` files into ordered task groups (from `## <heading>` lines) containing ordered tasks (from `- [ ]` / `- [x]` checkbox lines), preserving each task's id prefix (e.g., `1.2`), description text, checked state, and source line number.

#### Scenario: Well-formed tasks file
- **WHEN** a `tasks.md` with two groups of two checkbox tasks each is parsed
- **THEN** the model contains two groups with two tasks each, in file order, with correct checked states and line numbers

#### Scenario: Non-checkbox content is preserved
- **WHEN** a `tasks.md` contains prose, blank lines, or unknown markdown between tasks
- **THEN** parsing succeeds, tasks are still extracted, and the unknown lines are retained in the model verbatim (so writes never destroy them)

#### Scenario: Malformed file degrades gracefully
- **WHEN** a `tasks.md` contains no recognizable checkbox lines
- **THEN** the change is still listed, with zero tasks and a flag indicating tasks could not be parsed

### Requirement: Report artifact pipeline status
The system SHALL report, for each change, which schema artifacts (proposal, specs, design, tasks) exist on disk, so the UI can render the artifact pipeline without invoking the OpenSpec CLI.

#### Scenario: Partially complete change
- **WHEN** a change directory contains `proposal.md` and `design.md` but no `tasks.md` and no `specs/` files
- **THEN** the model reports proposal and design as present and specs and tasks as absent

### Requirement: Derive change lifecycle status from files only
The system SHALL derive each change's lifecycle status exclusively from on-disk state: **Archived** if under `changes/archive/`; otherwise **Complete** if the change has at least one task and all checkboxes are checked; otherwise **In Progress** if at least one checkbox is checked or any artifact beyond proposal.md exists; otherwise **Draft**. The system MUST NOT persist its own status metadata.

#### Scenario: All tasks checked
- **WHEN** a change's `tasks.md` has 5 tasks, all checked
- **THEN** its derived status is Complete

#### Scenario: Mixed checkboxes
- **WHEN** a change's `tasks.md` has 3 checked and 2 unchecked tasks
- **THEN** its derived status is In Progress

#### Scenario: Proposal only
- **WHEN** a change contains only `proposal.md`
- **THEN** its derived status is Draft
