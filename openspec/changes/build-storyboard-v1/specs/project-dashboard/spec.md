# Spec: project-dashboard

## ADDED Requirements

### Requirement: Lifecycle board of changes
The system SHALL display registered projects' changes as cards in lifecycle columns (Draft, In Progress, Complete, Archived), where each card shows the change name, its project (when multiple projects are registered), artifact pipeline status, and task progress as a checked/total count (e.g., `12/20`).

#### Scenario: Changes grouped by derived status
- **WHEN** a registered project has one Draft change and one In Progress change
- **THEN** each appears as a card in its corresponding column with name and task progress visible

#### Scenario: Card answers "where is it" without opening
- **WHEN** the user views the board
- **THEN** every card shows phase and progress without requiring hover, click, or drill-in

### Requirement: Multi-project overview and filtering
The system SHALL show changes from all registered projects on one board, visually attributed to their project, and SHALL let the user filter the board to a single project.

#### Scenario: Two projects registered
- **WHEN** two projects are registered and both have active changes
- **THEN** the board shows all changes with per-project attribution, and selecting one project filters the board to it

### Requirement: Empty states that teach
The system SHALL render instructive empty states: with no registered projects, the board explains how to register one; with a registered project containing no changes, the board suggests creating a change with the OpenSpec workflow (e.g., `/opsx:propose`).

#### Scenario: No projects registered
- **WHEN** the user opens the app with an empty registry
- **THEN** the main view explains what Storyboard does and offers the register-project action as the single primary call to action

#### Scenario: Project with no changes
- **WHEN** a registered project's `openspec/changes/` is empty
- **THEN** the board shows a message suggesting how to create a first change instead of a blank area

### Requirement: Navigation to change detail
The system SHALL open the change detail view when a change card is activated by mouse click or by keyboard (Enter on a focused card).

#### Scenario: Keyboard activation
- **WHEN** the user focuses a card with Tab/arrow keys and presses Enter
- **THEN** the change detail view for that change opens
