import { createContext, useContext } from 'react'

export const DocumentHeadingContext = createContext(null)

// Only document previews own a scrollable heading destination. Chat keeps DSH's link behavior.
export function DocumentLocalLink({ file, hash, children, settled, fallback, className }) {
  const navigate = useContext(DocumentHeadingContext)
  if (!navigate) return fallback
  // Authored local links in a file page can open as soon as the page is read.
  if (!hash) return settled
  let heading
  try { heading = decodeURIComponent(hash) } catch { return fallback }
  return <button type="button" className={className} onClick={() => navigate(file, heading)}>{children}</button>
}

export function revealHeading(root, fragment) {
  if (!root || !fragment) return
  const used = new Set()
  for (const element of root.querySelectorAll('[data-document-markdown] :is(h1,h2,h3,h4,h5,h6)')) {
    const base = element.textContent.trim().toLowerCase().replace(/[^\p{L}\p{M}\p{N}\s_-]/gu, '').replace(/\s/g, '-')
    let slug = base
    for (let suffix = 1; used.has(slug); suffix++) slug = base + '-' + suffix
    used.add(slug)
    if (slug !== fragment.toLowerCase() && element.textContent.trim() !== fragment) continue
    element.tabIndex = -1
    element.focus({ preventScroll: true })
    element.scrollIntoView({ block: 'start', inline: 'nearest' })
    return true
  }
}
