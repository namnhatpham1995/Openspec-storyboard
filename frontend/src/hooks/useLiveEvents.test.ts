import { describe, expect, it } from 'vitest'
import { formatRelativeTime } from './useLiveEvents'

describe('formatRelativeTime', () => {
  const now = new Date('2026-07-19T12:00:00Z').getTime()

  it.each([
    ['2026-07-19T11:59:40Z', 'just now'],
    ['2026-07-19T11:55:00Z', '5m ago'],
    ['2026-07-19T09:00:00Z', '3h ago'],
    ['2026-07-17T12:00:00Z', '2d ago'],
  ])('formats %s as %s', (timestamp, expected) => {
    expect(formatRelativeTime(timestamp, now)).toBe(expected)
  })
})
