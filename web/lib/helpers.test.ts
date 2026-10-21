import { describe, expect, it } from 'vitest'
import { brandTheme, contrastRatio, darkenUntilContrast, parseHex, readableTextOn, safeColor } from '@/lib/contrast'
import { valueAt } from '@/lib/count'
import { formatDate } from '@/lib/format'
import { buildSrcSet } from '@/lib/images'
import { expiredTranslateCookies, readTargetLanguage, targetFromCookieValue, translateCookie } from '@/lib/language'
import { safeHref } from '@/lib/links'
import { isInternalNavigation } from '@/lib/navigation'
import { pageHref, pageItems, parsePageParam, totalPages } from '@/lib/pagination'
import { buildRobots } from '@/lib/robots'
import { shareTargets } from '@/lib/share'
import { buildSitemapEntries } from '@/lib/sitemap'
import { autoplayDelayMs, indexFromScroll, nextIndex, shouldAutoplay } from '@/lib/slider'
import { embedSrc } from '@/lib/video'
import type { PublicImage } from '@/lib/types'

describe('color contrast', () => {
  it('computes known ratios', () => {
    expect(contrastRatio([0, 0, 0], [255, 255, 255])).toBeCloseTo(21, 0)
  })

  it('picks readable text for dark and light backgrounds', () => {
    expect(readableTextOn('#0B2545')).toBe('#FFFFFF')
    expect(readableTextOn('#F5E663')).toBe('#0B1220')
  })

  it('derives an AA on-accent color for the default teal accent', () => {
    const theme = brandTheme('#0B2545', '#13A89E')
    const onAccent = parseHex(theme.onAccent)!
    expect(contrastRatio(onAccent, parseHex(theme.accent)!)).toBeGreaterThanOrEqual(4.5)
  })

  it('darkens an accent until it is AA readable as text on white', () => {
    const text = darkenUntilContrast('#13A89E', '#FFFFFF')
    expect(contrastRatio(parseHex(text)!, [255, 255, 255])).toBeGreaterThanOrEqual(4.5)
  })

  it('rejects invalid colors and expands shorthand', () => {
    expect(safeColor('red; background:url(x)', '#0B2545')).toBe('#0B2545')
    expect(safeColor('#abc', '#000000')).toBe('#AABBCC')
  })
})

describe('pagination', () => {
  it('parses page params defensively', () => {
    expect(parsePageParam('3')).toBe(3)
    expect(parsePageParam(['2', '5'])).toBe(2)
    expect(parsePageParam('-1')).toBe(1)
    expect(parsePageParam('abc')).toBe(1)
    expect(parsePageParam(undefined)).toBe(1)
  })

  it('computes total pages with a minimum of one', () => {
    expect(totalPages(0, 9)).toBe(1)
    expect(totalPages(19, 9)).toBe(3)
  })

  it('collapses long ranges with gaps', () => {
    expect(pageItems(1, 3)).toEqual([1, 2, 3])
    expect(pageItems(5, 10)).toEqual([1, 'gap', 4, 5, 6, 'gap', 10])
  })

  it('keeps page one on the bare path', () => {
    expect(pageHref('/news', 1)).toBe('/news')
    expect(pageHref('/news', 2)).toBe('/news?page=2')
  })
})

describe('language switcher cookie logic', () => {
  it('reads the target language from the cookie header', () => {
    expect(readTargetLanguage('a=1; googtrans=/en/id; b=2', 'en')).toBe('id')
    expect(readTargetLanguage('googtrans=%2Fen%2Ftl', 'en')).toBe('tl')
  })

  it('treats the source language, missing and malformed cookies as untranslated', () => {
    expect(readTargetLanguage('googtrans=/en/en', 'en')).toBeNull()
    expect(readTargetLanguage('a=1', 'en')).toBeNull()
    expect(targetFromCookieValue('garbage', 'en')).toBeNull()
    expect(targetFromCookieValue('%E0%A4%A', 'en')).toBeNull()
  })

  it('serializes and expires the cookie', () => {
    expect(translateCookie('id', 'en')).toContain('googtrans=/en/id; path=/')
    expect(expiredTranslateCookies('example.com')).toHaveLength(3)
    expect(expiredTranslateCookies('localhost')).toHaveLength(1)
  })
})

describe('hero autoplay logic', () => {
  const base = { count: 3, seconds: 5, reducedMotion: false, hovered: false, focused: false, hidden: false }

  it('plays when idle', () => {
    expect(shouldAutoplay(base)).toBe(true)
  })

  it.each([
    ['hover', { hovered: true }],
    ['focus', { focused: true }],
    ['hidden tab', { hidden: true }],
    ['reduced motion', { reducedMotion: true }],
    ['single slide', { count: 1 }],
    ['zero interval', { seconds: 0 }],
  ])('pauses on %s', (_name, override) => {
    expect(shouldAutoplay({ ...base, ...override })).toBe(false)
  })

  it('wraps indexes and clamps the delay', () => {
    expect(nextIndex(2, 3, 1)).toBe(0)
    expect(nextIndex(0, 3, -1)).toBe(2)
    expect(autoplayDelayMs(1)).toBe(3000)
    expect(indexFromScroll(850, 400, 3)).toBe(2)
  })
})

describe('misc helpers', () => {
  it('counts up to the final value', () => {
    expect(valueAt(100, 0, 1000)).toBe(0)
    expect(valueAt(100, 1000, 1000)).toBe(100)
    expect(valueAt(100, 500, 1000)).toBeGreaterThan(50)
  })

  it('formats dates as DD-MMM-YYYY in UTC', () => {
    expect(formatDate('2026-03-05T23:30:00Z')).toBe('05-Mar-2026')
    expect(formatDate('')).toBe('')
  })

  it('builds srcset from sources sorted by width', () => {
    const image: PublicImage = { url: 'o', width: 1600, height: 900, alt: '', mime_type: '', sources: [{ url: 'b', width: 960, height: 1 }, { url: 'a', width: 480, height: 1 }] }
    expect(buildSrcSet(image)).toBe('a 480w, b 960w')
    expect(buildSrcSet({ ...image, sources: [] })).toBeUndefined()
  })

  it('only allows safe link schemes', () => {
    expect(safeHref('/career')).toBe('/career')
    expect(safeHref('https://a.com')).toBe('https://a.com')
    expect(safeHref('javascript:alert(1)')).toBe('')
    expect(safeHref('//evil.com')).toBe('')
  })

  it('builds embed URLs on privacy-friendly hosts and rejects bad ids', () => {
    expect(embedSrc({ provider: 'youtube', video_id: 'abcDEF12345' })).toContain('youtube-nocookie.com')
    expect(embedSrc({ provider: 'vimeo', video_id: '12345678' })).toContain('player.vimeo.com')
    expect(embedSrc({ provider: 'youtube', video_id: '"><script>' })).toBe('')
  })

  it('creates share links for the four networks', () => {
    expect(shareTargets('https://a.com/x', 'Hi there').map((t) => t.key)).toEqual(['x', 'facebook', 'linkedin', 'whatsapp'])
  })

  it('detects internal navigations only', () => {
    const location = { origin: 'http://a.com', pathname: '/', search: '' }
    const click = { defaultPrevented: false, button: 0, metaKey: false, ctrlKey: false, shiftKey: false, altKey: false }
    const anchor = (href: string, target = '') => ({ href, target, hasAttribute: () => false })
    expect(isInternalNavigation(click, anchor('http://a.com/news'), location)).toBe(true)
    expect(isInternalNavigation(click, anchor('http://a.com/#x'), location)).toBe(false)
    expect(isInternalNavigation(click, anchor('http://b.com/news'), location)).toBe(false)
    expect(isInternalNavigation(click, anchor('http://a.com/news', '_blank'), location)).toBe(false)
    expect(isInternalNavigation({ ...click, ctrlKey: true }, anchor('http://a.com/news'), location)).toBe(false)
  })

  it('disallows everything in robots when indexing is off', () => {
    expect(buildRobots(false, 'https://a.com')).toEqual({ rules: { userAgent: '*', disallow: '/' }, sitemap: 'https://a.com/sitemap.xml' })
    expect(buildRobots(true, 'https://a.com').rules).toMatchObject({ allow: '/' })
  })

  it('builds absolute sitemap URLs with lastmod and dedupes', () => {
    const entries = buildSitemapEntries(
      { pages: [{ path: '/about', lastmod: '2026-01-01T00:00:00Z' }], system_pages: [{ path: '/', lastmod: '2026-02-01T00:00:00Z' }], news: null, vacancies: [] },
      'https://a.com',
    )
    expect(entries.map((e) => e.url)).toEqual(['https://a.com/', 'https://a.com/about'])
    expect(entries[1]?.lastModified).toEqual(new Date('2026-01-01T00:00:00Z'))
  })
})
