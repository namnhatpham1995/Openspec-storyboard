package openspec

import "errors"

// ErrNotOpenSpecProject is returned by Discover when the given folder has
// no openspec/ directory.
var ErrNotOpenSpecProject = errors.New("not an openspec project: no openspec/ directory found")
