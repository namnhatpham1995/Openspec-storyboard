import type { PropsWithChildren } from 'react'
import { Link } from 'react-router-dom'

export function AppShell({ children }: PropsWithChildren) {
  return (
    <div className="app-shell">
      <header className="topbar">
        <Link className="brand" to="/" aria-label="Storyboard board">
          <span className="brand-mark" aria-hidden="true">S</span>
          <span>
            <strong>Storyboard</strong>
            <small>OpenSpec drafting table</small>
          </span>
        </Link>
        <div className="local-badge"><span aria-hidden="true" /> Local only</div>
      </header>
      <main>{children}</main>
      <footer className="footer-note">Files on disk are the source of truth.</footer>
    </div>
  )
}
