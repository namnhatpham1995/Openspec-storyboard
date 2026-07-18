package openspec

import (
	"reflect"
	"testing"
)

func TestParseTasksDoc(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    TasksDoc
	}{
		{
			name: "well-formed two groups of two tasks",
			content: "" +
				"## 1. Setup\n" +
				"\n" +
				"- [ ] 1.1 Install Go\n" +
				"- [x] 1.2 Init module\n" +
				"\n" +
				"## 2. Parser\n" +
				"\n" +
				"- [ ] 2.1 Define structs\n" +
				"- [ ] 2.2 Write tests\n",
			want: TasksDoc{
				Groups: []TaskGroup{
					{
						Heading: "1. Setup",
						Tasks: []Task{
							{ID: "1.1", Text: "Install Go", Checked: false, Line: 3},
							{ID: "1.2", Text: "Init module", Checked: true, Line: 4},
						},
					},
					{
						Heading: "2. Parser",
						Tasks: []Task{
							{ID: "2.1", Text: "Define structs", Checked: false, Line: 8},
							{ID: "2.2", Text: "Write tests", Checked: false, Line: 9},
						},
					},
				},
				Parseable: true,
			},
		},
		{
			name: "non-checkbox content between tasks is not lost",
			content: "" +
				"# tasks.md\n" +
				"\n" +
				"Some prose explaining the plan.\n" +
				"\n" +
				"## 1. Group\n" +
				"\n" +
				"A note before the first task.\n" +
				"- [ ] 1.1 Only task\n",
			want: TasksDoc{
				Groups: []TaskGroup{
					{
						Heading: "1. Group",
						Tasks: []Task{
							{ID: "1.1", Text: "Only task", Checked: false, Line: 8},
						},
					},
				},
				Parseable: true,
			},
		},
		{
			name: "no checkbox lines is unparseable but not an error",
			content: "" +
				"## 1. Group\n" +
				"\n" +
				"This file forgot to add any checkboxes.\n",
			want: TasksDoc{
				Groups:    []TaskGroup{{Heading: "1. Group"}},
				Parseable: false,
			},
		},
		{
			name: "task with no id prefix",
			content: "" +
				"## 1. Group\n" +
				"- [ ] Task with no numeric id\n",
			want: TasksDoc{
				Groups: []TaskGroup{
					{
						Heading: "1. Group",
						Tasks: []Task{
							{ID: "", Text: "Task with no numeric id", Checked: false, Line: 2},
						},
					},
				},
				Parseable: true,
			},
		},
		{
			name:    "CRLF line endings parse the same as LF",
			content: "## 1. Group\r\n- [x] 1.1 Done\r\n- [ ] 1.2 Not done\r\n",
			want: TasksDoc{
				Groups: []TaskGroup{
					{
						Heading: "1. Group",
						Tasks: []Task{
							{ID: "1.1", Text: "Done", Checked: true, Line: 2},
							{ID: "1.2", Text: "Not done", Checked: false, Line: 3},
						},
					},
				},
				Parseable: true,
			},
		},
		{
			name:    "empty file",
			content: "",
			want:    TasksDoc{Parseable: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTasksDoc([]byte(tt.content))

			if !reflect.DeepEqual(got.Groups, tt.want.Groups) {
				t.Errorf("Groups = %#v, want %#v", got.Groups, tt.want.Groups)
			}
			if got.Parseable != tt.want.Parseable {
				t.Errorf("Parseable = %v, want %v", got.Parseable, tt.want.Parseable)
			}
		})
	}
}

func TestParseTasksDoc_RawLinesPreservesContent(t *testing.T) {
	content := "## 1. Group\n\nA note.\n- [ ] 1.1 Task\n"

	got := ParseTasksDoc([]byte(content))

	want := []string{"## 1. Group", "", "A note.", "- [ ] 1.1 Task"}
	if !reflect.DeepEqual(got.RawLines, want) {
		t.Errorf("RawLines = %#v, want %#v", got.RawLines, want)
	}

	// The task's Line field must point at its actual position in RawLines.
	line := got.Groups[0].Tasks[0].Line
	if got.RawLines[line-1] != "- [ ] 1.1 Task" {
		t.Errorf("RawLines[%d] = %q, want the task's source line", line-1, got.RawLines[line-1])
	}
}
