import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getChangeDetail, updateArtifact } from '../api/client'
import type { ChangeDetail, FileVersion } from '../api/types'
import { ChangeDetailPage } from './ChangeDetailPage'

vi.mock('../api/client', async (importOriginal) => ({
  ...await importOriginal<typeof import('../api/client')>(),
  getChangeDetail: vi.fn(),
  updateArtifact: vi.fn(),
}))

const getChangeDetailMock = vi.mocked(getChangeDetail)
const updateArtifactMock = vi.mocked(updateArtifact)

const version: FileVersion = { modTime: '2026-01-01T00:00:00Z', hash: 'design-version' }
const detail: ChangeDetail = {
  name: 'demo', archived: false, status: 'in_progress',
  artifacts: { proposal: true, design: true, specs: true, tasks: true },
  tasks: { groups: [], parseable: true },
  artifactFiles: [
    { kind: 'proposal', path: 'proposal.md', content: '# Proposal', version: { modTime: '', hash: 'proposal-version' } },
    { kind: 'design', path: 'design.md', content: '# Design', version },
    { kind: 'spec', path: 'specs/capability/spec.md', content: '# Spec', version: { modTime: '', hash: 'spec-version' } },
    { kind: 'tasks', path: 'tasks.md', content: '## Tasks', version: { modTime: '', hash: 'tasks-version' } },
  ],
}

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/projects/project-1/changes/demo']}>
        <Routes><Route path="/projects/:projectID/changes/:name" element={<ChangeDetailPage />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('ChangeDetailPage artifact editing', () => {
  it('edits, cancels, and saves a selected design artifact', async () => {
    const user = userEvent.setup()
    getChangeDetailMock.mockResolvedValue(detail)
    updateArtifactMock.mockResolvedValue({
      artifact: { ...detail.artifactFiles[1], content: '# Edited design', version: { ...version, hash: 'new-design-version' } },
    })
    renderPage()

    await user.click(await screen.findByRole('tab', { name: 'design.md' }))
    await user.click(screen.getByRole('button', { name: 'Edit artifact' }))
    const editor = screen.getByRole('textbox', { name: 'design.md markdown' })
    await user.clear(editor)
    await user.type(editor, '# Discarded design')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.getByRole('heading', { name: 'Design' })).toBeInTheDocument()
    expect(updateArtifactMock).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'Edit artifact' }))
    await user.clear(screen.getByRole('textbox', { name: 'design.md markdown' }))
    await user.type(screen.getByRole('textbox', { name: 'design.md markdown' }), '# Edited design')
    await user.click(screen.getByRole('button', { name: 'Save artifact' }))

    await waitFor(() => expect(updateArtifactMock).toHaveBeenCalledWith(
      'project-1', 'demo', 'design.md', '# Edited design', version,
    ))
  })
})
