import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { taskCounts } from '../utils/taskCounts'
import { TaskProgress } from './TaskProgress'

const tasks = {
  parseable: true,
  groups: [{
    heading: 'One',
    tasks: [
      { id: '1.1', text: 'Done', checked: true, line: 1 },
      { id: '1.2', text: 'Next', checked: false, line: 2 },
    ],
  }],
}

describe('TaskProgress', () => {
  it('counts and announces completed tasks', () => {
    expect(taskCounts(tasks)).toEqual({ checked: 1, total: 2 })
    render(<TaskProgress tasks={tasks} />)
    expect(screen.getByLabelText('1 of 2 tasks complete')).toBeInTheDocument()
    expect(screen.getByText('1/2')).toBeInTheDocument()
  })
})
