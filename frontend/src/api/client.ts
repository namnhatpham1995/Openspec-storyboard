import type { ChangeDetail, FileVersion, Project, ToggleResult } from './types'

export class APIError extends Error {
  readonly status: number
  readonly code: string

  constructor(message: string, status: number, code = 'request_failed') {
    super(message)
    this.status = status
    this.code = code
  }
}

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { Accept: 'application/json', ...init?.headers },
  })
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as
      | { error?: { code?: string; message?: string } }
      | null
    throw new APIError(
      body?.error?.message ?? `Request failed (${response.status})`,
      response.status,
      body?.error?.code,
    )
  }
  return response.json() as Promise<T>
}

export const getCurrentProject = () => requestJSON<Project>('/api/projects/current')

export const getChangeDetail = (name: string) =>
  requestJSON<ChangeDetail>(`/api/changes/${encodeURIComponent(name)}`)

export const toggleTask = (changeName: string, taskID: string, version: FileVersion) =>
  requestJSON<ToggleResult>(
    `/api/changes/${encodeURIComponent(changeName)}/tasks/${encodeURIComponent(taskID)}/toggle`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ version }),
    },
  )
