import { afterEach, describe, expect, it, vi } from 'vitest'
import { addProject, APIError, getDirectories, getProjects, removeProject, toggleTask, updateProposal, updateTaskText } from './client'

afterEach(() => vi.restoreAllMocks())

describe('toggleTask', () => {
  it('posts the base version and returns the updated task', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      task: { id: '1.1', text: 'Task', checked: true, line: 2 },
      version: { modTime: '2026-01-01T00:00:01Z', hash: 'new' },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    const version = { modTime: '2026-01-01T00:00:00Z', hash: 'old' }
    const result = await toggleTask('project-1', 'demo', '1.1', version)

    expect(result.task.checked).toBe(true)
    expect(fetchMock).toHaveBeenCalledWith('/api/projects/project-1/changes/demo/tasks/1.1/toggle', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ version }),
    }))
  })

  it('exposes conflict status and code', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'file_conflict', message: 'file changed externally' },
    }), { status: 409, headers: { 'Content-Type': 'application/json' } }))

    await expect(toggleTask('project-1', 'demo', '1.1', { modTime: '', hash: 'old' })).rejects.toEqual(
      expect.objectContaining<Partial<APIError>>({ status: 409, code: 'file_conflict' }),
    )
  })
})

describe('text writes', () => {
  it('puts task text with its base version', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      task: { id: '1.1', text: 'Edited', checked: false, line: 2 },
      version: { modTime: '2026-01-01T00:00:01Z', hash: 'new' },
    }), { status: 200 }))
    const version = { modTime: '2026-01-01T00:00:00Z', hash: 'old' }

    await updateTaskText('project-1', 'demo', '1.1', 'Edited', version)

    expect(fetchMock).toHaveBeenCalledWith('/api/projects/project-1/changes/demo/tasks/1.1/text', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ text: 'Edited', version }),
    }))
  })

  it('puts raw proposal markdown with its base version', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      artifact: { kind: 'proposal', path: 'proposal.md', content: '# Edited', version: { modTime: '', hash: 'new' } },
    }), { status: 200 }))
    const version = { modTime: '2026-01-01T00:00:00Z', hash: 'old' }

    await updateProposal('project-1', 'demo', '# Edited', version)

    expect(fetchMock).toHaveBeenCalledWith('/api/projects/project-1/changes/demo/artifacts/proposal', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ content: '# Edited', version }),
    }))
  })
})

describe('project registry', () => {
  it('lists and adds projects', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({ projects: [] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        id: 'project-1', path: 'C:\\work\\demo', name: 'demo', connected: true, changes: [],
      }), { status: 201 }))

    expect((await getProjects()).projects).toEqual([])
    await addProject('C:\\work\\demo')

    expect(fetchMock).toHaveBeenLastCalledWith('/api/projects', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ path: 'C:\\work\\demo' }),
    }))
  })

  it('removes a project without expecting a JSON body', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 204 }))

    await removeProject('project-1')

    expect(fetchMock).toHaveBeenCalledWith('/api/projects/project-1', { method: 'DELETE' })
  })
})

describe('directory browser', () => {
  it('loads the default directory without a query', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      path: '/home/demo', parentPath: '/home', directories: [], locations: [],
    }), { status: 200 }))

    await getDirectories()

    expect(fetchMock).toHaveBeenCalledWith('/api/filesystem/directories', expect.objectContaining({
      headers: expect.objectContaining({ Accept: 'application/json' }),
    }))
  })

  it('encodes an explicit directory path', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      path: 'C:\\work & demos', directories: [], locations: [],
    }), { status: 200 }))

    await getDirectories('C:\\work & demos')

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/filesystem/directories?path=C%3A%5Cwork%20%26%20demos',
      expect.any(Object),
    )
  })
})
