import type { ChangeDetail, Project } from './types'

export class APIError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path, { headers: { Accept: 'application/json' } })
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as
      | { error?: { message?: string } }
      | null
    throw new APIError(body?.error?.message ?? `Request failed (${response.status})`, response.status)
  }
  return response.json() as Promise<T>
}

export const getCurrentProject = () => getJSON<Project>('/api/projects/current')

export const getChangeDetail = (name: string) =>
  getJSON<ChangeDetail>(`/api/changes/${encodeURIComponent(name)}`)
