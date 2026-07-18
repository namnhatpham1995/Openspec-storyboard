package openspec

// DeriveStatus computes a change's lifecycle status from on-disk state
// only, per the openspec-parsing spec:
//
//   - Archived, if the change lives under changes/archive/.
//   - Complete, if it has at least one task and every task is checked.
//   - InProgress, if at least one task is checked, or any artifact
//     beyond proposal.md exists (design, specs, or a tasks.md file).
//   - Draft, otherwise.
func DeriveStatus(archived bool, artifacts Artifacts, tasks TasksDoc) Status {
	if archived {
		return StatusArchived
	}

	total, checked := countTasks(tasks)

	if total > 0 && checked == total {
		return StatusComplete
	}
	if checked > 0 || artifacts.Design || artifacts.Specs || artifacts.Tasks {
		return StatusInProgress
	}
	return StatusDraft
}

func countTasks(tasks TasksDoc) (total, checked int) {
	for _, group := range tasks.Groups {
		for _, task := range group.Tasks {
			total++
			if task.Checked {
				checked++
			}
		}
	}
	return total, checked
}
