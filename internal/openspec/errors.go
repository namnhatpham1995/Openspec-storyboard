package openspec

import "errors"

// ErrNotOpenSpecProject is returned by Discover when the given folder has
// no openspec/ directory.
var ErrNotOpenSpecProject = errors.New("not an openspec project: no openspec/ directory found")

// ErrChangeNotFound is returned when a requested active or archived change
// does not exist in the project.
var ErrChangeNotFound = errors.New("openspec change not found")

// ErrTaskNotFound is returned when a requested task id does not identify
// exactly one task in the change's tasks.md file.
var ErrTaskNotFound = errors.New("openspec task not found")

// ErrConflict is returned when a write is based on a stale file version.
var ErrConflict = errors.New("file changed externally")

// ErrInvalidTaskLine is returned when line surgery is asked to modify a line
// that does not contain a recognizable task checkbox.
var ErrInvalidTaskLine = errors.New("line does not contain a task checkbox")
