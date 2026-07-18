# Spec: change-detail

## ADDED Requirements

### Requirement: Task groups with checkbox glyphs
The system SHALL display a change's tasks grouped exactly as in `tasks.md`, each task rendered with a monospace `[ ]` or `[x]` glyph reflecting its on-disk state, its id (e.g., `1.2`), and its description, in file order.

#### Scenario: Groups mirror the file
- **WHEN** the detail view opens for a change whose `tasks.md` has groups "1. Parser" and "2. Server"
- **THEN** both groups appear with their tasks in the same order as the file, with correct glyph states

### Requirement: Toggle a task from the UI
The system SHALL toggle a task's checkbox in `tasks.md` when its glyph is clicked or when the focused task receives Space, and reflect the new state only after the write succeeds. Toggling MUST be symmetric (clicking again reverts) and keyboard navigation (arrow keys between tasks) MUST be supported.

#### Scenario: Successful toggle
- **WHEN** the user clicks the `[ ]` glyph of task 1.2
- **THEN** the corresponding line in `tasks.md` becomes `[x]` on disk and the UI updates to show `[x]`

#### Scenario: Stale toggle rejected
- **WHEN** the file changed on disk after the UI last read it and the user toggles a task
- **THEN** no write occurs, the view reloads from disk, and the user is told the file changed externally

### Requirement: Artifact pipeline and rendered artifacts
The system SHALL show the change's artifact pipeline (proposal → specs → design → tasks) with per-artifact presence, and SHALL render each existing artifact's markdown read-only in the detail view.

#### Scenario: Viewing the proposal
- **WHEN** the user selects the proposal artifact of a change
- **THEN** its markdown renders in the detail view without modifying the file

### Requirement: Edit task and proposal text
The system SHALL allow editing a task's description text and the proposal's markdown from the UI, showing the affected source lines as editable plain text (not a rich-text projection), and saving via the markdown-writes capability.

#### Scenario: Edit a task description
- **WHEN** the user edits task 1.2's text and saves
- **THEN** only that task's line in `tasks.md` changes on disk, and the checkbox state and id prefix are preserved

#### Scenario: Cancel an edit
- **WHEN** the user starts an edit and cancels
- **THEN** no write occurs and the view continues to show on-disk content
