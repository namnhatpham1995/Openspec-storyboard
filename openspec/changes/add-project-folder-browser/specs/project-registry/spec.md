## ADDED Requirements

### Requirement: Browse for a project folder
The system SHALL provide a Browse action beside every project-folder path input that opens a keyboard-accessible directory navigator. The navigator SHALL allow the user to move among accessible local directories and platform-appropriate starting locations, SHALL display directories without displaying files, and MUST NOT modify filesystem contents while browsing.

#### Scenario: Open the directory navigator
- **WHEN** the user activates Browse from a project registration form
- **THEN** a modal directory navigator opens at the current valid directory path or a platform-appropriate default location

#### Scenario: Navigate directories
- **WHEN** the user opens a child directory, parent directory, breadcrumb, or starting location
- **THEN** the navigator displays that directory's absolute path and its accessible child directories in deterministic order

#### Scenario: Select a project folder
- **WHEN** the user chooses the displayed directory and confirms "Use this folder"
- **THEN** the navigator closes and the directory's absolute path is placed in the editable project-folder input without registering the project

#### Scenario: Cancel folder selection
- **WHEN** the user cancels the navigator or dismisses it with Escape
- **THEN** the navigator closes, focus returns to Browse, and the project-folder input remains unchanged

#### Scenario: Directory cannot be listed
- **WHEN** the selected directory is missing, invalid, or inaccessible
- **THEN** the navigator presents an accessible error, preserves the existing project-folder input, and allows the user to navigate elsewhere or cancel

#### Scenario: Directory contains no child directories
- **WHEN** the displayed directory has no accessible child directories
- **THEN** the navigator shows an empty-folder state and still allows the user to select the displayed directory, navigate elsewhere, or cancel

#### Scenario: Enter a project path manually
- **WHEN** the user types or pastes a project path instead of browsing
- **THEN** the existing registration flow accepts and validates that value unchanged
