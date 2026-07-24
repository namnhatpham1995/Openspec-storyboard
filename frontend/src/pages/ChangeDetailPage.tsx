import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import ReactMarkdown from 'react-markdown'
import { Link, useParams } from 'react-router-dom'
import { APIError, getChangeDetail, toggleTask, updateArtifact, updateTaskText } from '../api/client'
import type { ArtifactFile, FileVersion, Task } from '../api/types'
import { ArtifactPipeline } from '../components/ArtifactPipeline'
import { ErrorState } from '../components/ErrorState'
import { TaskProgress } from '../components/TaskProgress'
import { useArrowNavigation } from '../hooks/useArrowNavigation'

export function ChangeDetailPage() {
  const { projectID = '', name = '' } = useParams()
  const queryClient = useQueryClient()
  const detail = useQuery({
    queryKey: ['change', projectID, name],
    queryFn: () => getChangeDetail(projectID, name),
    enabled: Boolean(projectID && name),
  })
  const [selectedPath, setSelectedPath] = useState('')
  const [notice, setNotice] = useState<{ kind: 'conflict' | 'error'; message: string } | null>(null)
  const [writeReset, setWriteReset] = useState(0)

  const refreshFromDisk = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['change', projectID, name] }),
      queryClient.invalidateQueries({ queryKey: ['projects'] }),
    ])
  }
  const writeSucceeded = async () => {
    setNotice(null)
    await refreshFromDisk()
  }
  const writeFailed = async (error: unknown, fallback: string) => {
    if (error instanceof APIError && error.status === 409) {
      setNotice({ kind: 'conflict', message: 'File changed externally. Reloaded the latest version from disk.' })
      setWriteReset((current) => current + 1)
      await refreshFromDisk()
      return
    }
    setNotice({ kind: 'error', message: error instanceof Error ? error.message : fallback })
  }
  const externalConflict = () => {
    setNotice({ kind: 'conflict', message: 'File changed externally. Reloaded the latest version from disk.' })
    setWriteReset((current) => current + 1)
  }

  const toggle = useMutation({
    mutationFn: ({ taskID, version }: { taskID: string; version: FileVersion }) =>
      toggleTask(projectID, name, taskID, version),
    onSuccess: writeSucceeded,
    onError: (error) => writeFailed(error, 'Could not update the task.'),
  })
  const taskText = useMutation({
    mutationFn: ({ taskID, text, version }: { taskID: string; text: string; version: FileVersion }) =>
      updateTaskText(projectID, name, taskID, text, version),
    onSuccess: writeSucceeded,
    onError: (error) => writeFailed(error, 'Could not update the task text.'),
  })
  const artifactText = useMutation({
    mutationFn: ({ path, content, version }: { path: string; content: string; version: FileVersion }) =>
      updateArtifact(projectID, name, path, content, version),
    onSuccess: writeSucceeded,
    onError: (error) => writeFailed(error, 'Could not update the artifact.'),
  })

  const artifactFiles = detail.data?.artifactFiles
  const selectedArtifact = artifactFiles?.find((artifact) => artifact.path === selectedPath) ?? artifactFiles?.[0]
  const tasksVersion = artifactFiles?.find((artifact) => artifact.path === 'tasks.md')?.version

  useEffect(() => {
    if (!selectedPath && artifactFiles?.[0]) setSelectedPath(artifactFiles[0].path)
  }, [artifactFiles, selectedPath])

  if (detail.isPending) return <DetailSkeleton />
  if (detail.isError) return <ErrorState title="Change not found" message={detail.error.message} />

  const taskWritePending = toggle.isPending || taskText.isPending

  return (
    <article className="detail-page">
      <Link className="back-link" to="/">← Back to board</Link>
      <header className="detail-heading">
        <div>
          <p className="eyebrow">{detail.data.status.replace('_', ' ')}</p>
          <h1>{detail.data.name}</h1>
        </div>
        <TaskProgress tasks={detail.data.tasks} />
      </header>

      <section className="detail-pipeline">
        <div><p className="eyebrow">Artifact pipeline</p><ArtifactPipeline artifacts={detail.data.artifacts} /></div>
        <p className="disk-note"><span aria-hidden="true">↻</span> Read directly from disk</p>
      </section>

      {notice && (
        <div className={`write-notice write-notice--${notice.kind}`} role="alert">
          <span aria-hidden="true">{notice.kind === 'conflict' ? '↻' : '!'}</span>
          <span>{notice.message}</span>
          <button type="button" onClick={() => setNotice(null)} aria-label="Dismiss notification">×</button>
        </div>
      )}

      <div className="detail-grid">
        <section className="task-panel" aria-labelledby="tasks-title">
          <div className="panel-heading"><p className="eyebrow">Plan</p><h2 id="tasks-title">Tasks</h2></div>
          {(detail.data.tasks.groups ?? []).map((group) => (
            <div className="task-group" key={group.heading}>
              <h3>{group.heading}</h3>
              <ul>
                {(group.tasks ?? []).map((task) => (
                  <TaskRow
                    key={`${task.id}-${task.line}-${writeReset}`}
                    task={task}
                    version={tasksVersion}
                    disabled={!tasksVersion || taskWritePending}
                    pendingToggle={toggle.isPending && toggle.variables?.taskID === task.id}
                    pendingText={taskText.isPending && taskText.variables?.taskID === task.id}
                    onToggle={(taskID) => tasksVersion && toggle.mutate({ taskID, version: tasksVersion })}
                    onSave={(taskID, text, version) => taskText.mutateAsync({ taskID, text, version }).then(() => undefined)}
                    onExternalChange={externalConflict}
                  />
                ))}
              </ul>
            </div>
          ))}
          {!detail.data.tasks.parseable && <p className="panel-empty">No checkbox tasks could be parsed from tasks.md.</p>}
        </section>

        <section className="artifact-panel" aria-labelledby="artifact-title">
          <div className="panel-heading"><p className="eyebrow">Source</p><h2 id="artifact-title">Artifacts</h2></div>
          <div className="artifact-tabs" role="tablist" aria-label="Artifact files">
            {(artifactFiles ?? []).map((artifact) => (
              <button
                key={artifact.path}
                type="button"
                role="tab"
                aria-selected={selectedArtifact?.path === artifact.path}
                onClick={() => setSelectedPath(artifact.path)}
              >
                {artifact.path}
              </button>
            ))}
          </div>
          {selectedArtifact ? (
            <MarkdownArtifact
              key={`${selectedArtifact.path}-${writeReset}`}
              artifact={selectedArtifact}
              saving={artifactText.isPending}
              onSave={selectedArtifact.path !== 'tasks.md'
                ? (content, version) => artifactText.mutateAsync({ path: selectedArtifact.path, content, version }).then(() => undefined)
                : undefined}
              onExternalChange={externalConflict}
            />
          ) : <p className="panel-empty">No artifact files found.</p>}
        </section>
      </div>
    </article>
  )
}

function TaskRow({
  task,
  version,
  disabled,
  pendingToggle,
  pendingText,
  onToggle,
  onSave,
  onExternalChange,
}: {
  task: Task
  version?: FileVersion
  disabled: boolean
  pendingToggle: boolean
  pendingText: boolean
  onToggle: (taskID: string) => void
  onSave: (taskID: string, text: string, version: FileVersion) => Promise<void>
  onExternalChange: () => void
}) {
  const onArrowKeyDown = useArrowNavigation('[data-task-row]')
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(task.text)
  const [editBaseText, setEditBaseText] = useState(task.text)
  const [editVersion, setEditVersion] = useState<FileVersion | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (editing) inputRef.current?.select()
  }, [editing])
  useEffect(() => {
    if (!editing) setDraft(task.text)
    if (editing && !pendingText && task.text !== editBaseText) {
      setDraft(task.text)
      setEditing(false)
      onExternalChange()
    }
  }, [editBaseText, editing, onExternalChange, pendingText, task.text])

  const startEditing = () => {
    if (!version) return
    setDraft(task.text)
    setEditBaseText(task.text)
    setEditVersion(version)
    setEditing(true)
  }

  const cancel = () => {
    setDraft(task.text)
    setEditing(false)
  }
  const save = async () => {
    if (!task.id || !editVersion || draft === editBaseText) {
      setEditing(false)
      return
    }
    try {
      await onSave(task.id, draft, editVersion)
      setEditing(false)
    } catch {
      // The mutation displays the API or conflict notice; keep the draft open.
    }
  }
  const onKeyDown = (event: React.KeyboardEvent<HTMLLIElement>) => {
    if (editing) return
    if (event.key === ' ' && !disabled && task.id) {
      event.preventDefault()
      onToggle(task.id)
      return
    }
    if (event.key === 'Enter' && !disabled && task.id) {
      event.preventDefault()
      startEditing()
      return
    }
    onArrowKeyDown(event)
  }

  return (
    <li className={`${task.checked ? 'is-checked' : ''} ${editing ? 'is-editing' : ''}`} tabIndex={editing ? -1 : 0} data-task-row onKeyDown={onKeyDown} aria-busy={pendingToggle || pendingText}>
      <button
        className="task-glyph mono"
        type="button"
        tabIndex={-1}
        disabled={disabled || !task.id || editing}
        onClick={() => task.id && onToggle(task.id)}
        aria-label={`${task.checked ? 'Mark incomplete' : 'Mark complete'}: task ${task.id || 'without id'}`}
      >
        {pendingToggle ? '[·]' : task.checked ? '[x]' : '[ ]'}
      </button>
      {task.id && <span className="task-id mono">{task.id}</span>}
      {editing ? (
        <div className="task-text-editor">
          <input
            ref={inputRef}
            value={draft}
            disabled={pendingText}
            aria-label={`Edit task ${task.id}`}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              event.stopPropagation()
              if (event.key === 'Escape') cancel()
              if (event.key === 'Enter') void save()
            }}
          />
          <button type="button" onClick={() => void save()} disabled={pendingText}>{pendingText ? 'Saving…' : 'Save'}</button>
          <button type="button" onClick={cancel} disabled={pendingText}>Cancel</button>
        </div>
      ) : (
        <button className="task-text-button" type="button" tabIndex={-1} disabled={disabled || !task.id} onClick={startEditing}>
          <span>{task.text || 'Empty task text'}</span><small>Edit</small>
        </button>
      )}
    </li>
  )
}

function MarkdownArtifact({
  artifact,
  saving,
  onSave,
  onExternalChange,
}: {
  artifact: ArtifactFile
  saving: boolean
  onSave?: (content: string, version: FileVersion) => Promise<void>
  onExternalChange: () => void
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(artifact.content)
  const [editVersion, setEditVersion] = useState(artifact.version)
  const previousHash = useRef(artifact.version.hash)

  useEffect(() => {
    if (previousHash.current === artifact.version.hash) return
    previousHash.current = artifact.version.hash
    setDraft(artifact.content)
    if (editing && !saving) onExternalChange()
    setEditing(false)
  }, [artifact.content, artifact.version.hash, editing, onExternalChange, saving])

  const startEditing = () => {
    setDraft(artifact.content)
    setEditVersion(artifact.version)
    setEditing(true)
  }

  const cancel = () => {
    setDraft(artifact.content)
    setEditing(false)
  }
  const save = async () => {
    if (!onSave || draft === artifact.content) {
      setEditing(false)
      return
    }
    try {
      await onSave(draft, editVersion)
      setEditing(false)
    } catch {
      // The mutation displays the API or conflict notice; keep the draft open.
    }
  }

  return (
    <div className="markdown-wrap">
      <div className="artifact-file-meta">
        <span className="mono">{artifact.path}</span>
        <span className="artifact-file-actions">
          <span title={artifact.version.hash}>rev {artifact.version.hash.slice(0, 7)}</span>
          {onSave && !editing && <button type="button" onClick={startEditing}>Edit artifact</button>}
        </span>
      </div>
      {editing ? (
        <div className="proposal-editor">
          <textarea
            value={draft}
            disabled={saving}
            aria-label={`${artifact.path} markdown`}
            spellCheck={false}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Escape') cancel()
            }}
          />
          <div className="editor-actions">
            <span>Raw Markdown · Esc cancels</span>
            <button type="button" onClick={cancel} disabled={saving}>Cancel</button>
            <button className="is-primary" type="button" onClick={() => void save()} disabled={saving}>{saving ? 'Saving…' : 'Save artifact'}</button>
          </div>
        </div>
      ) : (
        <div className="markdown-body"><ReactMarkdown>{artifact.content}</ReactMarkdown></div>
      )}
    </div>
  )
}

function DetailSkeleton() {
  return <section className="detail-page" aria-label="Loading change"><div className="skeleton skeleton--heading" /><div className="skeleton skeleton--detail" /></section>
}
