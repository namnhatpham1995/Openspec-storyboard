import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { getCurrentProject } from '../api/client'
import type { Change, ChangeStatus } from '../api/types'
import { ArtifactPipeline } from '../components/ArtifactPipeline'
import { ErrorState } from '../components/ErrorState'
import { TaskProgress } from '../components/TaskProgress'
import { useArrowNavigation } from '../hooks/useArrowNavigation'

const columns: Array<{ status: ChangeStatus; title: string; note: string }> = [
  { status: 'draft', title: 'Draft', note: 'Ideas taking shape' },
  { status: 'in_progress', title: 'In progress', note: 'Work on the table' },
  { status: 'complete', title: 'Complete', note: 'Ready to archive' },
  { status: 'archived', title: 'Archived', note: 'Filed for reference' },
]

function ChangeCard({ change }: { change: Change }) {
  const onKeyDown = useArrowNavigation('[data-change-card]')
  return (
    <Link
      className="change-card"
      to={`/changes/${encodeURIComponent(change.name)}`}
      data-change-card
      onKeyDown={onKeyDown}
      aria-label={`${change.name}, ${change.status.replace('_', ' ')}`}
    >
      <div className="change-card__pin" aria-hidden="true" />
      <span className="change-card__phase">{change.status.replace('_', ' ')}</span>
      <h3>{change.name}</h3>
      <ArtifactPipeline artifacts={change.artifacts} compact />
      <TaskProgress tasks={change.tasks} />
    </Link>
  )
}

export function BoardPage() {
  const project = useQuery({ queryKey: ['project', 'current'], queryFn: getCurrentProject })

  if (project.isPending) return <BoardSkeleton />
  if (project.isError) {
    return <ErrorState message={project.error.message} />
  }

  const changes = project.data.changes ?? []
  if (changes.length === 0) {
    return (
      <section className="empty-state">
        <span className="empty-state__icon" aria-hidden="true">+</span>
        <p className="eyebrow">Clean sheet</p>
        <h1>No changes yet</h1>
        <p>Create your first OpenSpec change with <code>/opsx:propose</code>, then refresh this board.</p>
      </section>
    )
  }

  return (
    <section className="board-page">
      <div className="page-heading">
        <div>
          <p className="eyebrow">Current project</p>
          <h1>{projectName(project.data.root)}</h1>
        </div>
        <div className="project-meta">
          <span>{project.data.schemaName || 'OpenSpec'}</span>
          <span className="mono">{changes.length} {changes.length === 1 ? 'change' : 'changes'}</span>
        </div>
      </div>

      <div className="board" aria-label="Change lifecycle board">
        {columns.map((column) => {
          const cards = changes.filter((change) => change.status === column.status)
          return (
            <section className={`board-column board-column--${column.status}`} key={column.status}>
              <header>
                <div><h2>{column.title}</h2><p>{column.note}</p></div>
                <span className="column-count mono">{cards.length}</span>
              </header>
              <div className="card-stack">
                {cards.map((change) => <ChangeCard key={change.name} change={change} />)}
                {cards.length === 0 && <p className="column-empty">Nothing here yet</p>}
              </div>
            </section>
          )
        })}
      </div>
      <p className="keyboard-hint"><kbd>Tab</kbd> or arrow keys move between cards. <kbd>Enter</kbd> opens a change.</p>
    </section>
  )
}

function projectName(root: string) {
  const pieces = root.replaceAll('\\', '/').split('/').filter(Boolean)
  return pieces.at(-1) ?? 'OpenSpec project'
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
