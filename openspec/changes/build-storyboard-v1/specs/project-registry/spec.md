# Spec: project-registry

## ADDED Requirements

### Requirement: Register and remove project folders
The system SHALL let the user register project folders (paths containing an `openspec/` directory) and remove them, persisting the registry to a single JSON config file in the OS user-config directory (`os.UserConfigDir()/storyboard/config.json`). The system MUST NOT write registry data anywhere inside user project folders.

#### Scenario: Register a valid project
- **WHEN** the user registers a folder that contains `openspec/`
- **THEN** it is added to the registry, persisted to the config file, and appears on the dashboard

#### Scenario: Register an invalid folder
- **WHEN** the user registers a folder with no `openspec/` directory
- **THEN** the registration is rejected with a message explaining what an OpenSpec project folder looks like

#### Scenario: Remove a project
- **WHEN** the user removes a registered project
- **THEN** it disappears from the dashboard and the config file, and no files inside the project folder are modified

### Requirement: Validate registered paths on load
The system SHALL validate each registered path at startup and when reloading; paths that no longer exist or no longer contain `openspec/` SHALL be shown as disconnected rather than silently dropped from the registry.

#### Scenario: Project folder moved or deleted
- **WHEN** the app starts and a registered path no longer exists
- **THEN** the dashboard shows that project as disconnected with its last-known name and an option to remove or relocate it

### Requirement: Corrupt or missing config is non-fatal
The system SHALL start with an empty registry when the config file is missing, and SHALL back up and reset the config file when it cannot be parsed, surfacing a notice in the UI instead of failing to launch.

#### Scenario: First launch
- **WHEN** the app is launched with no config file present
- **THEN** the app starts normally and shows the empty-state onboarding for registering a first project

#### Scenario: Corrupt config file
- **WHEN** the config file contains invalid JSON
- **THEN** the app renames it to a backup, starts with an empty registry, and informs the user
