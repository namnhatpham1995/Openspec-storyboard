interface Breadcrumb {
  label: string
  path: string
}

export function breadcrumbs(path: string): Breadcrumb[] {
  if (/^[A-Za-z]:\\/.test(path)) {
    const root = path.slice(0, 3)
    const parts = path.slice(3).split('\\').filter(Boolean)
    const crumbs: Breadcrumb[] = [{ label: root, path: root }]
    let current = root.replace(/\\$/, '')
    for (const part of parts) {
      current += `\\${part}`
      crumbs.push({ label: part, path: current })
    }
    return crumbs
  }
  if (path.startsWith('/')) {
    const crumbs: Breadcrumb[] = [{ label: '/', path: '/' }]
    let current = ''
    for (const part of path.split('/').filter(Boolean)) {
      current += `/${part}`
      crumbs.push({ label: part, path: current })
    }
    return crumbs
  }
  return [{ label: path, path }]
}
