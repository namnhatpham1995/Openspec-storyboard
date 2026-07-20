import { useCallback, useEffect, useRef, useState } from 'react'
import { getDirectories } from '../api/client'
import type { DirectoryListing } from '../api/types'
import { breadcrumbs } from '../utils/pathBreadcrumbs'

interface DirectoryBrowserDialogProps {
  initialPath: string
  onSelect: (path: string) => void
  onClose: () => void
}

export function DirectoryBrowserDialog({ initialPath, onSelect, onClose }: DirectoryBrowserDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const closeButtonRef = useRef<HTMLButtonElement>(null)
  const returnFocusRef = useRef<HTMLElement | null>(null)
  const [listing, setListing] = useState<DirectoryListing | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const loadDirectory = useCallback(async (path = '') => {
    setLoading(true)
    setError('')
    try {
      const next = await getDirectories(path)
      setListing(next)
      return true
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Could not read the directory.')
      return false
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    returnFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    closeButtonRef.current?.focus()
    let active = true
    const loadInitialDirectory = async () => {
      const requested = initialPath.trim()
      const loaded = await loadDirectory(requested)
      if (active && requested && !loaded) {
        await loadDirectory()
      }
    }
    void loadInitialDirectory()
    return () => {
      active = false
      returnFocusRef.current?.focus()
    }
  }, [initialPath, loadDirectory])

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onClose()
        return
      }
      if (event.key !== 'Tab' || !dialogRef.current) return
      const focusable = Array.from(dialogRef.current.querySelectorAll<HTMLElement>(
        'button:not([disabled]), [href], input:not([disabled]), [tabindex]:not([tabindex="-1"])',
      ))
      if (focusable.length === 0) return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  const closeFromBackdrop = (event: React.MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) onClose()
  }

  return (
    <div className="directory-browser-backdrop" onMouseDown={closeFromBackdrop}>
      <div
        aria-labelledby="directory-browser-title"
        aria-modal="true"
        className="directory-browser"
        ref={dialogRef}
        role="dialog"
      >
        <header className="directory-browser__header">
          <div>
            <p className="eyebrow">Local filesystem</p>
            <h2 id="directory-browser-title">Choose a project folder</h2>
          </div>
          <button aria-label="Close folder browser" onClick={onClose} ref={closeButtonRef} type="button">×</button>
        </header>

        {listing && (
          <nav aria-label="Starting locations" className="directory-browser__locations">
            {listing.locations.map((location) => (
              <button disabled={loading} key={location.path} onClick={() => void loadDirectory(location.path)} type="button">
                {location.name}
              </button>
            ))}
          </nav>
        )}

        {listing && (
          <nav aria-label="Current directory" className="directory-browser__breadcrumbs">
            {breadcrumbs(listing.path).map((crumb, index) => (
              <span key={crumb.path}>
                {index > 0 && <span aria-hidden="true">/</span>}
                <button disabled={loading} onClick={() => void loadDirectory(crumb.path)} type="button">{crumb.label}</button>
              </span>
            ))}
          </nav>
        )}

        <div aria-live="polite" className="directory-browser__status">
          {loading && <span>Loading folders…</span>}
          {!loading && listing && <span className="mono">{listing.path}</span>}
        </div>
        {error && <p className="directory-browser__error" role="alert">{error}</p>}

        <div className="directory-browser__directories">
          {listing?.parentPath && (
            <button className="directory-row directory-row--parent" disabled={loading} onClick={() => void loadDirectory(listing.parentPath)} type="button">
              <span aria-hidden="true">↑</span><span>Parent folder</span>
            </button>
          )}
          {!loading && listing?.directories.map((directory) => (
            <button className="directory-row" disabled={loading} key={directory.path} onClick={() => void loadDirectory(directory.path)} type="button">
              <span aria-hidden="true">□</span><span>{directory.name}</span>
            </button>
          ))}
          {!loading && listing && listing.directories.length === 0 && (
            <p className="directory-browser__empty">This folder has no child directories.</p>
          )}
        </div>

        <footer className="directory-browser__actions">
          <button onClick={onClose} type="button">Cancel</button>
          <button
            className="is-primary"
            disabled={!listing || loading}
            onClick={() => { if (listing) onSelect(listing.path) }}
            type="button"
          >
            Use this folder
          </button>
        </footer>
      </div>
    </div>
  )
}
