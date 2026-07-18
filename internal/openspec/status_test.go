package openspec

import "testing"

func TestDeriveStatus(t *testing.T) {
	allChecked := TasksDoc{Groups: []TaskGroup{{Tasks: []Task{
		{Checked: true}, {Checked: true}, {Checked: true}, {Checked: true}, {Checked: true},
	}}}}
	mixedChecked := TasksDoc{Groups: []TaskGroup{{Tasks: []Task{
		{Checked: true}, {Checked: true}, {Checked: true}, {Checked: false}, {Checked: false},
	}}}}
	noTasks := TasksDoc{}

	tests := []struct {
		name      string
		archived  bool
		artifacts Artifacts
		tasks     TasksDoc
		want      Status
	}{
		{
			name:      "all tasks checked is complete",
			artifacts: Artifacts{Proposal: true, Tasks: true},
			tasks:     allChecked,
			want:      StatusComplete,
		},
		{
			name:      "mixed checkboxes is in progress",
			artifacts: Artifacts{Proposal: true, Tasks: true},
			tasks:     mixedChecked,
			want:      StatusInProgress,
		},
		{
			name:      "proposal only is draft",
			artifacts: Artifacts{Proposal: true},
			tasks:     noTasks,
			want:      StatusDraft,
		},
		{
			name:      "design present with no checked tasks is in progress",
			artifacts: Artifacts{Proposal: true, Design: true},
			tasks:     noTasks,
			want:      StatusInProgress,
		},
		{
			name:      "empty tasks.md present with zero tasks is in progress, not complete",
			artifacts: Artifacts{Proposal: true, Tasks: true},
			tasks:     TasksDoc{Parseable: false},
			want:      StatusInProgress,
		},
		{
			name:      "archived wins regardless of task state",
			archived:  true,
			artifacts: Artifacts{Proposal: true, Tasks: true},
			tasks:     allChecked,
			want:      StatusArchived,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveStatus(tt.archived, tt.artifacts, tt.tasks)
			if got != tt.want {
				t.Errorf("DeriveStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
