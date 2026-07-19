import type { TasksDoc } from '../api/types'
import { taskCounts } from '../utils/taskCounts'

export function TaskProgress({ tasks }: { tasks: TasksDoc }) {
  const { checked, total } = taskCounts(tasks)
  const percentage = total === 0 ? 0 : Math.round((checked / total) * 100)
  return (
    <div className="task-progress" aria-label={`${checked} of ${total} tasks complete`}>
      <div className="task-progress__track"><span style={{ width: `${percentage}%` }} /></div>
      <span className="mono">{checked}/{total}</span>
    </div>
  )
}
