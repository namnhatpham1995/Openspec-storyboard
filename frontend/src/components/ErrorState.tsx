export function ErrorState({ title = 'Could not load the board', message }: { title?: string; message: string }) {
  return (
    <section className="empty-state empty-state--error" role="alert">
      <span className="empty-state__icon" aria-hidden="true">!</span>
      <p className="eyebrow">Something went off the page</p>
      <h1>{title}</h1>
      <p>{message}</p>
      <button type="button" onClick={() => window.location.reload()}>Try again</button>
    </section>
  )
}
