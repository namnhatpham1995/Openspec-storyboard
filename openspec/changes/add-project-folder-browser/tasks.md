## 1. Directory Browsing Domain

- [ ] 1.1 Add an internal directory-browser package with response models and a lister that requires absolute paths, resolves the current and parent paths, returns directory entries only, and sorts them deterministically
- [ ] 1.2 Add platform-specific starting-location discovery for the user home, Unix filesystem root, and available Windows logical drives without adding a runtime dependency
- [ ] 1.3 Add table-driven package tests for default location, child and parent navigation, file filtering, symlinked directories, empty directories, relative paths, missing paths, and inaccessible paths

## 2. Read-Only Filesystem API

- [ ] 2.1 Add an injectable directory-listing seam to the server and register `GET /api/filesystem/directories` with optional absolute `path` handling and no-cache responses
- [ ] 2.2 Map invalid, missing, non-directory, and inaccessible paths to safe JSON API errors while logging useful server-side details
- [ ] 2.3 Add HTTP tests for the default response, explicit navigation, starting locations, deterministic directory-only results, rejected relative paths, and listing errors

## 3. Frontend Directory Navigator

- [ ] 3.1 Add TypeScript directory response types and a tested API client function for loading the default or requested directory
- [ ] 3.2 Build an accessible, lazily loaded folder-browser dialog with starting locations, breadcrumbs and parent navigation, child-directory buttons, loading, empty, and recoverable error states
- [ ] 3.3 Add component tests for opening at the typed path with default fallback, navigating, confirming a selection, cancelling without changes, Escape handling, focus restoration, empty folders, and API errors
- [ ] 3.4 Add Browse beside the shared project-path input, keep manual editing and explicit form submission unchanged, and style the controls and dialog responsively in the drafting-table design language

## 4. Validation and Embedded Assets

- [ ] 4.1 Format the Go changes and run the complete Go test suite
- [ ] 4.2 Run frontend lint, unit tests, and the production build, then commit the regenerated `frontend/dist` assets used by `go:embed`
- [ ] 4.3 Verify the Storyboard command cross-compiles for Windows, macOS, and Linux and manually smoke-test browse, cancel, path editing, and registration in the served application
