import { useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useState } from 'react'
import type { Activity, LiveEvent } from '../api/live'

const activityLimit = 20

export function useLiveEvents() {
  const queryClient = useQueryClient()
  const [activities, setActivities] = useState<Activity[]>([])
  const [connected, setConnected] = useState(false)

  const refreshQueries = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['projects'] }),
      queryClient.invalidateQueries({ queryKey: ['project', 'current'] }),
      queryClient.invalidateQueries({
        predicate: (query) => query.queryKey[0] === 'change',
      }),
    ])
  }, [queryClient])

  useEffect(() => {
    const source = new EventSource('/api/events')

    source.onopen = () => setConnected(true)
    source.onerror = () => setConnected(false)
    source.onmessage = (event) => {
      let update: LiveEvent
      try {
        update = JSON.parse(event.data) as LiveEvent
      } catch {
        return
      }

      if (update.type === 'ready') {
        setConnected(true)
        void refreshQueries()
        return
      }
      if (update.type !== 'project_changed') return

      if (update.activities?.length) {
        const incoming = update.activities.map((activity) => update.projectName
          ? { ...activity, message: `${update.projectName} · ${activity.message}` }
          : activity)
        setActivities((current) => [...incoming, ...current].slice(0, activityLimit))
      }
      void refreshQueries()
    }

    return () => source.close()
  }, [refreshQueries])

  const manualRefresh = useCallback(async () => {
    await refreshQueries()
    setActivities((current) => [{
      message: 'Manual refresh requested',
      file: '',
      action: 'refresh',
      timestamp: new Date().toISOString(),
    }, ...current].slice(0, activityLimit))
  }, [refreshQueries])

  return { activities, connected, manualRefresh }
}

export function formatRelativeTime(timestamp: string, now = Date.now()) {
  const elapsed = Math.max(0, now - new Date(timestamp).getTime())
  const seconds = Math.floor(elapsed / 1000)
  if (seconds < 45) return 'just now'

  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`

  const days = Math.floor(hours / 24)
  return `${days}d ago`
}
