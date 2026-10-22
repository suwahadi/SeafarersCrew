import { describe, expect, it } from 'vitest'
import { applyFavicon, DEFAULT_FAVICON, faviconLink } from './favicon'

describe('faviconLink', () => {
  it('uses the website favicon when one is set', () => {
    expect(faviconLink({ url: 'https://api.example.com/media/fav.webp', mime_type: 'image/webp' })).toEqual({ href: 'https://api.example.com/media/fav.webp', type: 'image/webp' })
  })

  it('falls back to the bundled favicon when none is set or loaded', () => {
    expect(faviconLink(null)).toEqual(DEFAULT_FAVICON)
    expect(faviconLink(undefined)).toEqual(DEFAULT_FAVICON)
  })
})

describe('applyFavicon', () => {
  it('updates the existing icon link instead of adding another one', () => {
    document.head.innerHTML = '<link rel="icon" type="image/svg+xml" href="/favicon.svg" />'
    applyFavicon(document, { href: 'https://x/fav.png', type: 'image/png' })
    const links = document.querySelectorAll('link[rel~="icon"]')
    expect(links).toHaveLength(1)
    expect(links[0]?.getAttribute('href')).toBe('https://x/fav.png')
    expect(links[0]?.getAttribute('type')).toBe('image/png')
  })

  it('creates the link when the page has none', () => {
    document.head.innerHTML = ''
    applyFavicon(document, DEFAULT_FAVICON)
    expect(document.querySelector('link[rel~="icon"]')?.getAttribute('href')).toBe('/favicon.svg')
  })
})
