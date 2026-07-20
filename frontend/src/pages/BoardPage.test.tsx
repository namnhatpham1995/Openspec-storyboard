import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getDirectories } from '../api/client'
import { ProjectForm } from './BoardPage'

vi.mock('../api/client', async (importOriginal) => ({
  ...await importOriginal<typeof import('../api/client')>(),
  getDirectories: vi.fn(),
}))

const getDirectoriesMock = vi.mocked(getDirectories)

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

describe('ProjectForm', () => {
  it('fills the editable path without submitting when a folder is selected', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    getDirectoriesMock.mockResolvedValue({
      path: 'C:\\selected-project',
      directories: [],
      locations: [{ name: 'Home', path: 'C:\\Users\\demo' }],
    })

    function Harness() {
      const [path, setPath] = useState('')
      return <ProjectForm error="" onSubmit={onSubmit} path={path} pending={false} setPath={setPath} />
    }

    render(<Harness />)
    await user.click(screen.getByRole('button', { name: 'Browse…' }))
    await user.click(await screen.findByRole('button', { name: 'Use this folder' }))

    expect(screen.getByLabelText('Project folder')).toHaveValue('C:\\selected-project')
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('continues to support manual entry and explicit submission', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    function Harness() {
      const [path, setPath] = useState('')
      return <ProjectForm error="" onSubmit={onSubmit} path={path} pending={false} setPath={setPath} />
    }

    render(<Harness />)
    await user.type(screen.getByLabelText('Project folder'), '/work/manual')
    await user.click(screen.getByRole('button', { name: 'Register project' }))

    expect(screen.getByLabelText('Project folder')).toHaveValue('/work/manual')
    expect(onSubmit).toHaveBeenCalledOnce()
  })
})
