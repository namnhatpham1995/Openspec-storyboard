import type { TasksDoc } from '../api/types'

export function taskCounts(tasks: TasksDoc) {
  const allTasks = (tasks.groups ?? []).flatMap((group) => group.tasks ?? [])
  return {
    checked: allTasks.filter((task) => task.checked).length,
    total: allTasks.length,
  }
}
