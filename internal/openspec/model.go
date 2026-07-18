package openspec

// Status is a change's lifecycle stage, derived entirely from files on
// disk (see DeriveStatus). Storyboard never stores its own status.
type Status string

const (
	StatusDraft      Status = "draft"
	StatusInProgress Status = "in_progress"
	StatusComplete   Status = "complete"
	StatusArchived   Status = "archived"
)

// Artifacts records which schema artifacts exist for a change, without
// needing the openspec CLI to answer the question.
type Artifacts struct {
	Proposal bool `json:"proposal"`
	Design   bool `json:"design"`
	Specs    bool `json:"specs"`
	Tasks    bool `json:"tasks"`
}

// Task is one checkbox line from a tasks.md file.
type Task struct {
	// ID is the task's numeric prefix, e.g. "1.2". Empty if the line had
	// no recognizable id.
	ID string `json:"id"`
	// Text is the description following the id (or the whole line after
	// the checkbox, when there is no id).
	Text string `json:"text"`
	// Checked reflects "- [x]" vs "- [ ]".
	Checked bool `json:"checked"`
	// Line is the 1-based line number in the source file, so callers can
	// locate the exact line to splice on write (see design.md decision D5).
	Line int `json:"line"`
}

// TaskGroup is one "## heading" section of a tasks.md file and the tasks
// under it, in file order.
type TaskGroup struct {
	Heading string `json:"heading"`
	Tasks   []Task `json:"tasks"`
}

// TasksDoc is the parsed form of a tasks.md file.
type TasksDoc struct {
	Groups []TaskGroup `json:"groups"`
	// Parseable is false when the file exists but contains no
	// recognizable "- [ ]"/"- [x]" lines at all. It is only meaningful
	// when the change's Artifacts.Tasks is true; a missing tasks.md is a
	// normal (Draft) state, not a parse failure.
	Parseable bool `json:"parseable"`
	// RawLines is the source file split into lines (1-based line N is
	// RawLines[N-1]), kept so unrecognized content is never dropped by
	// parsing. It is not part of the JSON wire format; the write path
	// (design.md decision D5) edits the original file's raw bytes
	// directly rather than reserializing from this struct.
	RawLines []string `json:"-"`
}

// Change is one OpenSpec change directory: a proposal plus whatever
// artifacts (design, specs, tasks) exist alongside it.
type Change struct {
	Name      string    `json:"name"`
	Archived  bool      `json:"archived"`
	Artifacts Artifacts `json:"artifacts"`
	Tasks     TasksDoc  `json:"tasks"`
	Status    Status    `json:"status"`
}

// Project is the parsed read model of one OpenSpec project folder.
type Project struct {
	// Root is the project folder path as given to Discover.
	Root string `json:"root"`
	// SchemaName is read from openspec/config.yaml; empty if the file is
	// absent or has no "schema:" line.
	SchemaName string   `json:"schemaName"`
	Changes    []Change `json:"changes"`
}
