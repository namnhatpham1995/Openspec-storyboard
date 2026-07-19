import { useEffect, useState } from 'react'
import type { Activity } from '../api/live'
import { formatRelativeTime } from '../hooks/useLiveEvents'

interface ActivityStripProps {
  activities: Activity[]
  connected: boolean
  onRefresh: () => Promise<void>
}

export function ActivityStrip({ activities, connected, onRefresh }: ActivityStripProps) {
  const [, setClock] = useState(Date.now())
  const [refreshing, setRefreshing] = useState(false)

  useEffect(() => {
    const timer = window.setInterval(() => setClock(Date.now()), 30_000)
    return () => window.clearInterval(timer)
  }, [])

  async function refresh() {
    setRefreshing(true)
    try {
      await onRefresh()
    } finally {
      setRefreshing(false)
    }
  }

  return (
    <aside className="activity-strip" aria-label="Live activity">
      <div className="activity-strip__heading">
        <span className={`activity-status ${connected ? 'is-connected' : ''}`} aria-hidden="true" />
        <strong>Live activity</strong>
        <span className="activity-connection">{connected ? 'Watching files' : 'Reconnecting'}</span>
        <button type="button" onClick={() => void refresh()} disabled={refreshing}>
          {refreshing ? 'Refreshing…' : 'Refresh'}
        </button>
      </div>
      <ol className="activity-list" aria-live="polite">
        {activities.length === 0 ? (
          <li className="activity-empty">External edits will appear here.</li>
        ) : activities.slice(0, 5).map((activity, index) => (
          <li key={`${activity.timestamp}-${activity.message}-${index}`}>
            <span>{activity.message}</span>
            <time dateTime={activity.timestamp}>{formatRelativeTime(activity.timestamp)}</time>
          </li>
        ))}
      </ol>
    </aside>
  )
}
