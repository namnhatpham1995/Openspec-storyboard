import type { Artifacts } from '../api/types'

const stages: Array<[keyof Artifacts, string]> = [
  ['proposal', 'Proposal'],
  ['specs', 'Specs'],
  ['design', 'Design'],
  ['tasks', 'Tasks'],
]

export function ArtifactPipeline({ artifacts, compact = false }: { artifacts: Artifacts; compact?: boolean }) {
  return (
    <ol className={`artifact-pipeline${compact ? ' artifact-pipeline--compact' : ''}`} aria-label="Artifact pipeline">
      {stages.map(([key, label]) => (
        <li key={key} className={artifacts[key] ? 'is-present' : ''}>
          <span aria-hidden="true">{artifacts[key] ? '●' : '○'}</span>
          {!compact && label}
          {compact && <span className="sr-only">{label}: {artifacts[key] ? 'present' : 'missing'}</span>}
        </li>
      ))}
    </ol>
  )
}
