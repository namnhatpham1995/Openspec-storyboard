## Context

The project registration form currently contains an editable text input and sends its value to `POST /api/projects`. Because Storyboard runs as a localhost Go server and serves its UI in a normal browser, a browser-native directory upload control cannot provide the absolute filesystem path required by the registry. The selection UI therefore needs a small read-only backend API that can enumerate directories for the local user.

The solution must preserve Storyboard's single-binary, cross-platform distribution, avoid writing during browsing, fit both first-launch and add-project flows, and remain usable by keyboard and assistive technology.

## Goals / Non-Goals

**Goals:**

- Let a user navigate local directories and place a selected absolute path into the existing project-folder input.
- Keep typing or pasting a path fully supported.
- Make browsing read-only, explicit, cancellable, and independent from project registration.
- Support Windows, macOS, and Linux without a browser extension, runtime installation, or external dialog program.
- Provide deterministic backend and frontend seams for focused tests.

**Non-Goals:**

- Replacing the existing OpenSpec-project validation or registration endpoint.
- Automatically registering a folder as soon as it is selected.
- Showing files, file contents, permissions, or other filesystem metadata not needed for navigation.
- Creating, renaming, deleting, or moving directories.
- Reproducing an operating system's native file-picker appearance.

## Decisions

### D1: Use a Storyboard directory navigator instead of a browser upload input or native OS dialog

The Browse button will open an accessible modal rendered by the React application. The modal requests one directory level at a time from the Go server, shows breadcrumb/parent navigation and platform-appropriate starting locations, and lets the user choose the currently displayed directory.

An `<input type="file" webkitdirectory>` was rejected because browsers intentionally omit a usable absolute path. Native dialog libraries and shelling out to `zenity`, `osascript`, or PowerShell were rejected because they add platform dependencies, complicate cross-compilation, or are unavailable on some installations.

### D2: Add one read-only directory-listing API

Add `GET /api/filesystem/directories` with an optional absolute `path` query parameter. With no path, the service starts at the user's home directory. A successful response contains the canonical current path, an optional parent path, sorted child directories, and navigation locations such as home, filesystem root, or Windows drive roots. Each child contains only a display name and absolute path.

The handler rejects relative paths, distinguishes invalid/not-found/inaccessible paths with safe API errors, returns no file entries, performs no writes, and does not enable cross-origin access or caching. Directory enumeration belongs in a small internal package behind an interface/function seam so server tests can use deterministic fixtures without reading the developer machine.

Starting-location discovery uses OS-specific files: home and `/` on Unix-like systems, and home plus available logical drive roots on Windows. The existing `golang.org/x/sys` module can supply the Windows drive mask; no new runtime dependency is introduced.

### D3: Navigate lazily, one directory level at a time

The frontend will fetch children only when a location is opened rather than recursively loading an entire tree. This bounds response size and latency on large disks and avoids touching unrelated subtrees. The dialog shows a loading state during navigation, an empty state for folders without child directories, and a recoverable inline error that leaves prior navigation available.

The current input value is used as the initial location when it is an absolute, listable directory; otherwise the browser falls back to the server's default location without overwriting the input.

### D4: Selection only updates the controlled input

Choosing "Use this folder" calls the existing `setPath` callback, closes the dialog, and returns focus to Browse. It does not call `POST /api/projects`. Cancel, Escape, and backdrop dismissal close the dialog without changing the input. The user can then edit the selected path or submit the existing form, which remains responsible for checking that the folder contains `openspec/`.

### D5: Treat accessibility as part of the component contract

The dialog has an accessible name, modal semantics, initial focus, contained tab navigation, Escape handling, and focus restoration. Directory rows are real buttons with visible focus states; current location, loading, and error changes are announced. The Browse button is adjacent to the input and remains present in both onboarding and later add-project forms because both use `ProjectForm`.

## Risks / Trade-offs

- [Directory names and absolute paths are exposed to the local UI] → Return directories only, keep the endpoint same-origin and loopback-only, require absolute paths, and avoid file contents and write operations.
- [Listing a slow or unavailable volume can block a request] → Enumerate only the selected directory, represent loading in the UI, and recover from per-request errors without clearing the current path input.
- [Platform roots and path syntax differ] → Isolate root discovery behind OS-specific files and cover path behavior with platform-neutral package tests plus targeted build checks.
- [Symlinked directories can navigate outside a displayed hierarchy] → Treat accessible directory symlinks as normal navigation targets and return the canonical path so the selected registry value is unambiguous.
- [A custom navigator is less familiar than the native picker] → Follow common file-picker conventions: locations, breadcrumbs, parent navigation, double-click/Enter to open, explicit selection, and cancellation.

## Migration Plan

Add the read-only backend route first, then the API client and dialog, and finally integrate Browse into the shared registration form. Rebuild the committed embedded frontend assets after tests pass. No registry or project-data migration is required. Rollback consists of removing the route and UI controls; existing typed paths and registry files remain compatible.

## Open Questions

None. The first implementation intentionally uses the current folder as the selection target and does not add directory creation or favorites.
