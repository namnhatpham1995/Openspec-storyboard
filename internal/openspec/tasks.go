package openspec

import (
	"regexp"
	"strings"
)

// groupHeadingPattern matches a task-group heading, e.g. "## 1. Setup".
// The whole line after "## " becomes the group's Heading.
var groupHeadingPattern = regexp.MustCompile(`^##\s+(.+?)\s*$`)

// taskLinePattern matches a checkbox line, e.g. "- [ ] 1.2 Do the thing"
// or "- [x] Do the thing" (an id prefix is optional).
var taskLinePattern = regexp.MustCompile(`^-\s\[([ xX])\]\s+(?:(\d+(?:\.\d+)*)\s+)?(.*?)\s*$`)

// ParseTasksDoc parses the contents of a tasks.md file into task groups.
//
// Parsing is intentionally lenient: any line that isn't a "## heading" or
// a "- [ ]"/"- [x]" checkbox is simply not part of the structured model.
// It is never an error and never drops content from RawLines, matching
// the openspec-parsing spec's requirement that unknown markdown survive
// parsing untouched.
func ParseTasksDoc(content []byte) TasksDoc {
	// Normalize CRLF to LF for splitting only; RawLines keeps the exact
	// original text per line (see the split below), and Task.Line still
	// indexes correctly because the number of lines is unaffected.
	rawLines := splitLines(string(content))

	var groups []TaskGroup
	var current *TaskGroup
	taskCount := 0

	for i, line := range rawLines {
		if m := groupHeadingPattern.FindStringSubmatch(line); m != nil {
			groups = append(groups, TaskGroup{Heading: m[1]})
			current = &groups[len(groups)-1]
			continue
		}

		if m := taskLinePattern.FindStringSubmatch(line); m != nil {
			task := Task{
				ID:      m[2],
				Text:    m[3],
				Checked: strings.EqualFold(m[1], "x"),
				Line:    i + 1,
			}
			if current == nil {
				groups = append(groups, TaskGroup{})
				current = &groups[len(groups)-1]
			}
			current.Tasks = append(current.Tasks, task)
			taskCount++
		}
	}

	return TasksDoc{
		Groups:    groups,
		Parseable: taskCount > 0,
		RawLines:  rawLines,
	}
}

// splitLines splits s into lines without its line terminators, handling
// both "\n" and "\r\n". The result always has at least one element
// (matching strings.Split's behavior for content with no trailing newline).
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	normalized := strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	// A trailing newline in the source produces one extra empty element
	// from strings.Split; drop it so line numbers stay 1:1 with content.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
