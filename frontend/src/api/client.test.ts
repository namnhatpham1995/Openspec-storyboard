import { afterEach, describe, expect, it, vi } from 'vitest'
import { APIError, toggleTask } from './client'

afterEach(() => vi.restoreAllMocks())

describe('toggleTask', () => {
  it('posts the base version and returns the updated task', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      task: { id: '1.1', text: 'Task', checked: true, line: 2 },
      version: { modTime: '2026-01-01T00:00:01Z', hash: 'new' },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    const version = { modTime: '2026-01-01T00:00:00Z', hash: 'old' }
    const result = await toggleTask('demo', '1.1', version)

    expect(result.task.checked).toBe(true)
    expect(fetchMock).toHaveBeenCalledWith('/api/changes/demo/tasks/1.1/toggle', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ version }),
    }))
  })

  it('exposes conflict status and code', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'file_conflict', message: 'file changed externally' },
    }), { status: 409, headers: { 'Content-Type': 'application/json' } }))

    await expect(toggleTask('demo', '1.1', { modTime: '', hash: 'old' })).rejects.toEqual(
      expect.objectContaining<Partial<APIError>>({ status: 409, code: 'file_conflict' }),
    )
  })
})
