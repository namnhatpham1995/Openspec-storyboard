// Package watch watches a project's openspec/ directory for filesystem
// changes, debounces bursts of events, and produces re-parsed snapshots
// and human-readable activity diffs for live updates (design.md decision D6).
package watch
