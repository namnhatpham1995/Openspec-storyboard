# Project workflow

The following Git and pull-request workflow is mandatory for every implementation change in this repository, including code, configuration, documentation, and OpenSpec phases.

## Before implementation

1. Confirm the working tree is safe to switch.
2. Check out `main`.
3. Pull the latest updates from `origin/main` with fast-forward-only semantics.
4. Create a dedicated branch for the new implementation.

Do not start new implementation work from an older feature branch. If local changes prevent switching safely, preserve them and resolve the situation before implementation begins.

## After implementation

1. Run the relevant validation checks.
2. Commit only the intended implementation scope.
3. Push the implementation branch.
4. Open a pull request targeting `main` for every implementation change.
5. The pull request must be ready for review, never a draft.

For dependency-ordered OpenSpec phases, do not begin the next phase until the current phase's pull request is merged.
