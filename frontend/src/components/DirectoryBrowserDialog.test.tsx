import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getDirectories } from '../api/client'
import type { DirectoryListing } from '../api/types'
import { breadcrumbs } from '../utils/pathBreadcrumbs'
import { DirectoryBrowserDialog } from './DirectoryBrowserDialog'

vi.mock('../api/client', () => ({ getDirectories: vi.fn() }))

const getDirectoriesMock = vi.mocked(getDirectories)
const homeListing: DirectoryListing = {
  path: '/home/demo',
  parentPath: '/home',
  directories: [{ name: 'project', path: '/home/demo/project' }],
  locations: [{ name: 'Home', path: '/home/demo' }, { name: 'Filesystem', path: '/' }],
}

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

describe('DirectoryBrowserDialog', () => {
  it('opens at a typed path and confirms the displayed folder', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    getDirectoriesMock.mockResolvedValue(homeListing)

    render(<DirectoryBrowserDialog initialPath="/typed/project" onClose={vi.fn()} onSelect={onSelect} />)

    await waitFor(() => expect(getDirectoriesMock).toHaveBeenCalledWith('/typed/project'))
    await user.click(await screen.findByRole('button', { name: 'Use this folder' }))
    expect(onSelect).toHaveBeenCalledWith('/home/demo')
  })

  it('falls back to the default location when the typed path cannot be listed', async () => {
    getDirectoriesMock
      .mockRejectedValueOnce(new Error('Directory not found'))
      .mockResolvedValueOnce(homeListing)

    render(<DirectoryBrowserDialog initialPath="/missing" onClose={vi.fn()} onSelect={vi.fn()} />)

    await waitFor(() => {
      expect(getDirectoriesMock).toHaveBeenNthCalledWith(1, '/missing')
      expect(getDirectoriesMock).toHaveBeenNthCalledWith(2, '')
    })
    expect(await screen.findByText('/home/demo')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('navigates into a child and shows an empty-folder state', async () => {
    const user = userEvent.setup()
    getDirectoriesMock
      .mockResolvedValueOnce(homeListing)
      .mockResolvedValueOnce({ ...homeListing, path: '/home/demo/project', directories: [] })

    render(<DirectoryBrowserDialog initialPath="" onClose={vi.fn()} onSelect={vi.fn()} />)

    await user.click(await screen.findByRole('button', { name: 'project' }))
    await waitFor(() => expect(getDirectoriesMock).toHaveBeenLastCalledWith('/home/demo/project'))
    expect(await screen.findByText('This folder has no child directories.')).toBeInTheDocument()
  })

  it('preserves the last listing and reports a navigation error', async () => {
    const user = userEvent.setup()
    getDirectoriesMock
      .mockResolvedValueOnce(homeListing)
      .mockRejectedValueOnce(new Error('Folder is inaccessible'))

    render(<DirectoryBrowserDialog initialPath="" onClose={vi.fn()} onSelect={vi.fn()} />)

    await user.click(await screen.findByRole('button', { name: 'project' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Folder is inaccessible')
    expect(screen.getByText('/home/demo')).toBeInTheDocument()
  })

  it('cancels without selecting, handles Escape, and restores focus', async () => {
    const user = userEvent.setup()
    getDirectoriesMock.mockResolvedValue(homeListing)

    function Harness() {
      const [open, setOpen] = useState(false)
      const [selected, setSelected] = useState('unchanged')
      return (
        <>
          <button onClick={() => setOpen(true)} type="button">Browse</button>
          <output>{selected}</output>
          {open && (
            <DirectoryBrowserDialog
              initialPath=""
              onClose={() => setOpen(false)}
              onSelect={setSelected}
            />
          )}
        </>
      )
    }

    render(<Harness />)
    const browse = screen.getByRole('button', { name: 'Browse' })
    await user.click(browse)
    await user.click(await screen.findByRole('button', { name: 'Cancel' }))
    expect(screen.getByText('unchanged')).toBeInTheDocument()
    expect(browse).toHaveFocus()

    await user.click(browse)
    await screen.findByRole('dialog')
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(browse).toHaveFocus()
  })

  it('contains keyboard focus within the modal', async () => {
    const user = userEvent.setup()
    getDirectoriesMock.mockResolvedValue(homeListing)
    render(<DirectoryBrowserDialog initialPath="" onClose={vi.fn()} onSelect={vi.fn()} />)
    await screen.findByText('/home/demo')

    const close = screen.getByRole('button', { name: 'Close folder browser' })
    close.focus()
    await user.keyboard('{Shift>}{Tab}{/Shift}')
    expect(screen.getByRole('button', { name: 'Use this folder' })).toHaveFocus()
  })
})

describe('breadcrumbs', () => {
  it('builds Unix and Windows navigation paths', () => {
    expect(breadcrumbs('/work/demo')).toEqual([
      { label: '/', path: '/' },
      { label: 'work', path: '/work' },
      { label: 'demo', path: '/work/demo' },
    ])
    expect(breadcrumbs('C:\\work\\demo')).toEqual([
      { label: 'C:\\', path: 'C:\\' },
      { label: 'work', path: 'C:\\work' },
      { label: 'demo', path: 'C:\\work\\demo' },
    ])
  })
})
