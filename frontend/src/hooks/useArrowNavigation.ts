import type { KeyboardEvent } from 'react'

export function useArrowNavigation(selector: string) {
  return (event: KeyboardEvent<HTMLElement>) => {
    const direction = ['ArrowRight', 'ArrowDown'].includes(event.key)
      ? 1
      : ['ArrowLeft', 'ArrowUp'].includes(event.key)
        ? -1
        : 0
    if (direction === 0) return

    const items = Array.from(document.querySelectorAll<HTMLElement>(selector))
    const current = items.indexOf(event.currentTarget)
    const next = items[current + direction]
    if (next) {
      event.preventDefault()
      next.focus()
    }
  }
}
