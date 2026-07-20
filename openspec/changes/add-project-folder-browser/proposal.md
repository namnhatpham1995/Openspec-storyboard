## Why

Registering a project currently requires users to find and type or paste an exact filesystem path. A built-in folder browser will make first-time setup and adding projects faster and less error-prone while keeping manual path entry available.

## What Changes

- Add a Browse control beside the project-folder path field wherever a project can be registered.
- Open a Storyboard directory-tree dialog that lets the user navigate folders available to the local Storyboard process.
- Fill the project-folder path field with the selected directory without registering it automatically, so the user can review or edit the path before submitting.
- Provide clear loading, empty-folder, inaccessible-folder, cancellation, and directory-listing error states while preserving keyboard and screen-reader usability.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `project-registry`: Extend project registration with a directory-tree browser that populates the existing project-path input.

## Impact

- Go server: add a read-only local-filesystem directory browsing API and platform-aware starting locations.
- React frontend: add the Browse control and accessible directory-tree dialog to the existing project registration form.
- Tests: cover directory listing boundaries and errors, API responses, path population, cancellation, and keyboard interaction.
- Security and data: the browser is limited to directory metadata needed for navigation; browsing does not register a project or modify filesystem contents.
