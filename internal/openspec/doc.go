// Package openspec discovers and parses OpenSpec project directories into
// a read model of projects, changes, artifacts, task groups, and tasks.
//
// It has no dependency on HTTP, the filesystem watcher, or any other
// package in this module — it reads only through io/fs so it can be
// tested against in-memory fixtures (see design.md decision D8).
package openspec
