# Project workflow

The following Git and pull-request workflow is mandatory for every implementation change in this repository, including code, configuration, documentation, and OpenSpec phases.

## Before implementation

1. Confirm the working tree is safe to switch.
2. Check out `main`.
3. Pull the latest updates from `origin/main` with fast-forward-only semantics.
4. Create a dedicated branch for the new implementation.

Do not start new implementation work from an older feature branch. If local changes prevent switching safely, preserve them and resolve the situation before implementation begins.

## Commit and pull-request titles

Every commit title and pull-request title MUST begin with an appropriate Conventional Commit prefix. Never omit the prefix.

Allowed prefixes are `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`, `build:`, `ci:`, `perf:`, and `revert:`. An optional scope and breaking-change marker may be used, such as `feat(server):` or `refactor!:`. Choose the prefix that best describes the primary change instead of defaulting every change to `feat:`.

## After implementation

1. Run the relevant validation checks.
2. Commit only the intended implementation scope.
3. Push the implementation branch.
4. Open a pull request targeting `main` for every implementation change.
5. The pull request must be ready for review, never a draft.
6. Write a complete PR description covering changes, rationale, impact, and validation.

When merging, use the PR title as the commit title and the exact PR description as the merge commit body. Repository merge and squash defaults are configured to use `PR_TITLE` and `PR_BODY`.

For dependency-ordered OpenSpec phases, do not begin the next phase until the current phase's pull request is merged.
