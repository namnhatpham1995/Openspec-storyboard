import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import ReactMarkdown from 'react-markdown'
import { Link, useParams } from 'react-router-dom'
import { APIError, getChangeDetail, toggleTask } from '../api/client'
import type { ArtifactFile, FileVersion, Task } from '../api/types'
import { ArtifactPipeline } from '../components/ArtifactPipeline'
import { ErrorState } from '../components/ErrorState'
import { TaskProgress } from '../components/TaskProgress'
import { useArrowNavigation } from '../hooks/useArrowNavigation'

export function ChangeDetailPage() {
  const { name = '' } = useParams()
  const queryClient = useQueryClient()
  const detail = useQuery({
    queryKey: ['change', name],
    queryFn: () => getChangeDetail(name),
    enabled: Boolean(name),
  })
  const [selectedPath, setSelectedPath] = useState('')
  const [notice, setNotice] = useState<{ kind: 'conflict' | 'error'; message: string } | null>(null)

  const artifactFiles = detail.data?.artifactFiles
  const selectedArtifact = artifactFiles?.find((artifact) => artifact.path === selectedPath) ?? artifactFiles?.[0]
  const tasksVersion = artifactFiles?.find((artifact) => artifact.path === 'tasks.md')?.version
  const toggle = useMutation({
    mutationFn: ({ taskID, version }: { taskID: string; version: FileVersion }) =>
      toggleTask(name, taskID, version),
    onSuccess: async () => {
      setNotice(null)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['change', name] }),
        queryClient.invalidateQueries({ queryKey: ['project', 'current'] }),
      ])
    },
    onError: async (error) => {
      if (error instanceof APIError && error.status === 409) {
        setNotice({ kind: 'conflict', message: 'File changed externally. Reloaded the latest version from disk.' })
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: ['change', name] }),
          queryClient.invalidateQueries({ queryKey: ['project', 'current'] }),
        ])
        return
      }
      setNotice({ kind: 'error', message: error instanceof Error ? error.message : 'Could not update the task.' })
    },
  })

  useEffect(() => {
    if (!selectedPath && artifactFiles?.[0]) setSelectedPath(artifactFiles[0].path)
  }, [artifactFiles, selectedPath])

  if (detail.isPending) return <DetailSkeleton />
  if (detail.isError) return <ErrorState title="Change not found" message={detail.error.message} />

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
                    key={`${task.id}-${task.line}`}
                    task={task}
                    disabled={!tasksVersion || toggle.isPending}
                    pending={toggle.isPending && toggle.variables?.taskID === task.id}
                    onToggle={(taskID) => tasksVersion && toggle.mutate({ taskID, version: tasksVersion })}
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
          {selectedArtifact ? <MarkdownArtifact artifact={selectedArtifact} /> : <p className="panel-empty">No artifact files found.</p>}
        </section>
      </div>
    </article>
  )
}

function TaskRow({
  task,
  disabled,
  pending,
  onToggle,
}: {
  task: Task
  disabled: boolean
  pending: boolean
  onToggle: (taskID: string) => void
}) {
  const onArrowKeyDown = useArrowNavigation('[data-task-row]')
  const onKeyDown = (event: React.KeyboardEvent<HTMLLIElement>) => {
    if (event.key === ' ' && !disabled && task.id) {
      event.preventDefault()
      onToggle(task.id)
      return
    }
    onArrowKeyDown(event)
  }
  return (
    <li className={task.checked ? 'is-checked' : ''} tabIndex={0} data-task-row onKeyDown={onKeyDown} aria-busy={pending}>
      <button
        className="task-glyph mono"
        type="button"
        tabIndex={-1}
        disabled={disabled || !task.id}
        onClick={() => task.id && onToggle(task.id)}
        aria-label={`${task.checked ? 'Mark incomplete' : 'Mark complete'}: task ${task.id || 'without id'}`}
      >
        {pending ? '[·]' : task.checked ? '[x]' : '[ ]'}
      </button>
      {task.id && <span className="task-id mono">{task.id}</span>}
      <span>{task.text}</span>
    </li>
  )
}

function MarkdownArtifact({ artifact }: { artifact: ArtifactFile }) {
  return (
    <div className="markdown-wrap">
      <div className="artifact-file-meta">
        <span className="mono">{artifact.path}</span>
        <span title={artifact.version.hash}>rev {artifact.version.hash.slice(0, 7)}</span>
      </div>
      <div className="markdown-body"><ReactMarkdown>{artifact.content}</ReactMarkdown></div>
    </div>
  )
}

function DetailSkeleton() {
  return <section className="detail-page" aria-label="Loading change"><div className="skeleton skeleton--heading" /><div className="skeleton skeleton--detail" /></section>
}
