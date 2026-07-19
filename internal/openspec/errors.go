package openspec

import "errors"

// ErrNotOpenSpecProject is returned by Discover when the given folder has
// no openspec/ directory.
var ErrNotOpenSpecProject = errors.New("not an openspec project: no openspec/ directory found")

// ErrChangeNotFound is returned when a requested active or archived change
// does not exist in the project.
var ErrChangeNotFound = errors.New("openspec change not found")
