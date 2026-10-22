import type { BrandingFavicon } from '@/api/branding'

export const DEFAULT_FAVICON = { href: '/favicon.svg', type: 'image/svg+xml' } as const

export function faviconLink(favicon: BrandingFavicon | null | undefined): { href: string; type: string } {
  if (!favicon?.url) return DEFAULT_FAVICON
  return { href: favicon.url, type: favicon.mime_type || DEFAULT_FAVICON.type }
}

export function applyFavicon(doc: Document, link: { href: string; type: string }): void {
  let el = doc.querySelector<HTMLLinkElement>('link[rel~="icon"]')
  if (!el) {
    el = doc.createElement('link')
    el.rel = 'icon'
    doc.head.appendChild(el)
  }
  el.type = link.type
  el.href = link.href
}
