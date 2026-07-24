import type { ArtifactWriteResult, ChangeDetail, DirectoryListing, FileVersion, ProjectsResponse, RegisteredProject, TaskTextResult, ToggleResult } from './types'

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

export const getProjects = () => requestJSON<ProjectsResponse>('/api/projects')

export const getDirectories = (path = '') => requestJSON<DirectoryListing>(
  `/api/filesystem/directories${path ? `?path=${encodeURIComponent(path)}` : ''}`,
)

export const addProject = (path: string) => requestJSON<RegisteredProject>('/api/projects', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ path }),
})

export const removeProject = async (projectID: string) => {
  const response = await fetch(`/api/projects/${encodeURIComponent(projectID)}`, { method: 'DELETE' })
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { error?: { code?: string; message?: string } } | null
    throw new APIError(body?.error?.message ?? `Request failed (${response.status})`, response.status, body?.error?.code)
  }
}

export const getChangeDetail = (projectID: string, name: string) =>
  requestJSON<ChangeDetail>(`/api/projects/${encodeURIComponent(projectID)}/changes/${encodeURIComponent(name)}`)

export const toggleTask = (projectID: string, changeName: string, taskID: string, version: FileVersion) =>
  requestJSON<ToggleResult>(
    `/api/projects/${encodeURIComponent(projectID)}/changes/${encodeURIComponent(changeName)}/tasks/${encodeURIComponent(taskID)}/toggle`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ version }),
    },
  )

export const updateTaskText = (projectID: string, changeName: string, taskID: string, text: string, version: FileVersion) =>
  requestJSON<TaskTextResult>(
    `/api/projects/${encodeURIComponent(projectID)}/changes/${encodeURIComponent(changeName)}/tasks/${encodeURIComponent(taskID)}/text`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text, version }),
    },
  )

export const updateProposal = (projectID: string, changeName: string, content: string, version: FileVersion) =>
  requestJSON<ArtifactWriteResult>(
    `/api/projects/${encodeURIComponent(projectID)}/changes/${encodeURIComponent(changeName)}/artifacts/proposal`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content, version }),
    },
  )
