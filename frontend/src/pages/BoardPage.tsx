import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { addProject, getProjects, removeProject } from '../api/client'
import type { Change, ChangeStatus, RegisteredProject } from '../api/types'
import { ArtifactPipeline } from '../components/ArtifactPipeline'
import { DirectoryBrowserDialog } from '../components/DirectoryBrowserDialog'
import { ErrorState } from '../components/ErrorState'
import { TaskProgress } from '../components/TaskProgress'
import { useArrowNavigation } from '../hooks/useArrowNavigation'

const columns: Array<{ status: ChangeStatus; title: string; note: string }> = [
  { status: 'draft', title: 'Draft', note: 'Ideas taking shape' },
  { status: 'in_progress', title: 'In progress', note: 'Work on the table' },
  { status: 'complete', title: 'Complete', note: 'Ready to archive' },
  { status: 'archived', title: 'Archived', note: 'Filed for reference' },
]

interface ProjectChange {
  project: RegisteredProject
  change: Change
}

function ChangeCard({ project, change }: ProjectChange) {
  const onKeyDown = useArrowNavigation('[data-change-card]')
  return (
    <Link
      className="change-card"
      to={`/projects/${encodeURIComponent(project.id)}/changes/${encodeURIComponent(change.name)}`}
      data-change-card
      onKeyDown={onKeyDown}
      aria-label={`${change.name}, ${change.status.replace('_', ' ')}, ${project.name}`}
    >
      <div className="change-card__pin" aria-hidden="true" />
      <span className="change-card__phase">{change.status.replace('_', ' ')}</span>
      <h3>{change.name}</h3>
      <span className="change-card__project">{project.name}</span>
      <ArtifactPipeline artifacts={change.artifacts} compact />
      <TaskProgress tasks={change.tasks} />
    </Link>
  )
}

export function BoardPage() {
  const queryClient = useQueryClient()
  const projectsQuery = useQuery({ queryKey: ['projects'], queryFn: getProjects })
  const [filter, setFilter] = useState('all')
  const [showAdd, setShowAdd] = useState(false)
  const [path, setPath] = useState('')

  const add = useMutation({
    mutationFn: addProject,
    onSuccess: async (project) => {
      setPath('')
      setShowAdd(false)
      setFilter(project.id)
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
  const remove = useMutation({
    mutationFn: removeProject,
    onSuccess: async (_, projectID) => {
      if (filter === projectID) setFilter('all')
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })

  if (projectsQuery.isPending) return <BoardSkeleton />
  if (projectsQuery.isError) return <ErrorState message={projectsQuery.error.message} />

  const projects = projectsQuery.data.projects ?? []
  if (projects.length === 0) {
    return (
      <ProjectOnboarding
        path={path}
        setPath={setPath}
        pending={add.isPending}
        error={add.error instanceof Error ? add.error.message : ''}
        onSubmit={() => path.trim() && add.mutate(path.trim())}
      />
    )
  }

  const visibleProjects = projects.filter((project) => project.connected && (filter === 'all' || filter === project.id))
  const changes: ProjectChange[] = visibleProjects.flatMap((project) =>
    (project.changes ?? []).map((change) => ({ project, change })),
  )
  const disconnected = projects.filter((project) => !project.connected)

  return (
    <section className="board-page">
      <div className="page-heading">
        <div>
          <p className="eyebrow">All OpenSpec work</p>
          <h1>Project dashboard</h1>
        </div>
        <div className="project-meta">
          <span>{projects.length} {projects.length === 1 ? 'project' : 'projects'}</span>
          <span className="mono">{changes.length} {changes.length === 1 ? 'change' : 'changes'}</span>
        </div>
      </div>

      <div className="project-toolbar">
        <div className="project-filters" aria-label="Filter by project">
          <button type="button" className={filter === 'all' ? 'is-active' : ''} onClick={() => setFilter('all')}>All projects</button>
          {projects.map((project) => (
            <button
              type="button"
              className={filter === project.id ? 'is-active' : ''}
              key={project.id}
              onClick={() => setFilter(project.id)}
            >
              <span className={`project-dot ${project.connected ? 'is-connected' : ''}`} aria-hidden="true" />
              {project.name}
            </button>
          ))}
        </div>
        <button className="add-project-button" type="button" onClick={() => setShowAdd((current) => !current)}>
          {showAdd ? 'Close' : '+ Add project'}
        </button>
      </div>

      {showAdd && (
        <ProjectForm
          path={path}
          setPath={setPath}
          pending={add.isPending}
          error={add.error instanceof Error ? add.error.message : ''}
          onSubmit={() => path.trim() && add.mutate(path.trim())}
        />
      )}

      {disconnected.length > 0 && (
        <div className="disconnected-projects" aria-label="Disconnected projects">
          {disconnected.map((project) => (
            <div key={project.id}>
              <span><strong>{project.name}</strong> · folder unavailable</span>
              <code>{project.path}</code>
              <button type="button" disabled={remove.isPending} onClick={() => remove.mutate(project.id)}>Remove</button>
            </div>
          ))}
        </div>
      )}

      <div className="board" aria-label="Change lifecycle board">
        {columns.map((column) => {
          const cards = changes.filter(({ change }) => change.status === column.status)
          return (
            <section className={`board-column board-column--${column.status}`} key={column.status}>
              <header>
                <div><h2>{column.title}</h2><p>{column.note}</p></div>
                <span className="column-count mono">{cards.length}</span>
              </header>
              <div className="card-stack">
                {cards.map(({ project, change }) => <ChangeCard key={`${project.id}-${change.name}`} project={project} change={change} />)}
                {cards.length === 0 && <p className="column-empty">Nothing here yet</p>}
              </div>
            </section>
          )
        })}
      </div>

      <div className="board-footer-row">
        <p className="keyboard-hint"><kbd>Tab</kbd> or arrow keys move between cards. <kbd>Enter</kbd> opens a change.</p>
        {filter !== 'all' && (
          <button type="button" className="remove-filtered-project" disabled={remove.isPending} onClick={() => remove.mutate(filter)}>
            Remove selected project
          </button>
        )}
      </div>
    </section>
  )
}

export function ProjectForm({
  path,
  setPath,
  pending,
  error,
  onSubmit,
}: {
  path: string
  setPath: (path: string) => void
  pending: boolean
  error: string
  onSubmit: () => void
}) {
  const [browsing, setBrowsing] = useState(false)
  return (
    <form className="project-form" onSubmit={(event) => { event.preventDefault(); onSubmit() }}>
      <label htmlFor="project-path">Project folder</label>
      <div className="project-path-control">
        <input
          id="project-path"
          value={path}
          onChange={(event) => setPath(event.target.value)}
          placeholder="C:\\work\\my-openspec-project"
          autoFocus
        />
        <button
          aria-haspopup="dialog"
          className="browse-project-button"
          disabled={pending}
          onClick={() => setBrowsing(true)}
          type="button"
        >
          Browse…
        </button>
      </div>
      <button type="submit" disabled={pending || !path.trim()}>{pending ? 'Checking…' : 'Register project'}</button>
      {error && <p role="alert">{error}</p>}
      <small>The folder must contain an <code>openspec/</code> directory.</small>
      {browsing && (
        <DirectoryBrowserDialog
          initialPath={path}
          onClose={() => setBrowsing(false)}
          onSelect={(selectedPath) => { setPath(selectedPath); setBrowsing(false) }}
        />
      )}
    </form>
  )
}

function ProjectOnboarding(props: Parameters<typeof ProjectForm>[0]) {
  return (
    <section className="empty-state onboarding-state">
      <span className="empty-state__icon" aria-hidden="true">S</span>
      <p className="eyebrow">Welcome to Storyboard</p>
      <h1>Bring your first project to the table</h1>
      <p>Register a local OpenSpec folder to see its proposals and tasks. Storyboard reads the files in place and never copies your project.</p>
      <ProjectForm {...props} />
    </section>
  )
}

function BoardSkeleton() {
  return (
    <section className="board-page" aria-label="Loading board">
      <div className="skeleton skeleton--heading" />
      <div className="board">
        {columns.map((column) => <div className="skeleton skeleton--column" key={column.status} />)}
      </div>
    </section>
  )
}
