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

// ErrArtifactNotFound is returned when a requested editable artifact does not
// exist for the change.
var ErrArtifactNotFound = errors.New("openspec artifact not found")

// ErrConflict is returned when a write is based on a stale file version.
var ErrConflict = errors.New("file changed externally")

// ErrArchiveNameConflict is returned when an archive already contains the
// date-prefixed name that would be assigned to a change.
var ErrArchiveNameConflict = errors.New("archive change name already exists")

// ErrInvalidTaskLine is returned when line surgery is asked to modify a line
// that does not contain a recognizable task checkbox.
var ErrInvalidTaskLine = errors.New("line does not contain a task checkbox")

// ErrInvalidTaskText is returned when replacement text cannot fit on one task
// line without changing the surrounding markdown structure.
var ErrInvalidTaskText = errors.New("task text must be a single line")
