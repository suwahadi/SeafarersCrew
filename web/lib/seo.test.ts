import { describe, expect, it } from 'vitest'
import { breadcrumbJsonLd, newsArticleJsonLd, organizationJsonLd, serializeJsonLd } from '@/lib/jsonld'
import { applyTitlePattern, buildMetadata, canonicalUrl, homeTitle, resolveOgImage } from '@/lib/seo'
import type { NewsArticle, PublicImage, Site } from '@/lib/types'

const image = (url: string, width = 1600): PublicImage => ({
  url,
  mime_type: 'image/webp',
  width,
  height: 900,
  alt: 'alt',
  sources: [
    { url: `${url}-480`, width: 480, height: 270 },
    { url: `${url}-1200`, width: 1200, height: 675 },
    { url, width, height: 900 },
  ],
})

const site = {
  name: 'Seafarers Crew',
  tagline: 'Crewing',
  organization: { legal_name: 'PT SC', email: 'a@b.c', phone: '', address: 'Dock 1', country: 'ID' },
  branding: { logo_light: image('https://x/logo'), logo_dark: null },
  contact: { email: '', phones: null },
  social: { linkedin: 'https://l', facebook: '', instagram: '', youtube: '', x: '' },
  seo: { meta_title_pattern: '%s | Seafarers Crew', meta_description: 'Default description', default_og_image: image('https://x/default') },
} as unknown as Site

describe('seo helpers', () => {
  it('applies the title pattern', () => {
    expect(applyTitlePattern('%s | Seafarers Crew', 'About')).toBe('About | Seafarers Crew')
  })

  it('falls back to the bare title when the pattern has no placeholder', () => {
    expect(applyTitlePattern('Static', 'About')).toBe('About')
  })

  it('does not interpret replacement tokens inside the title', () => {
    expect(applyTitlePattern('%s!', 'Cost $& more')).toBe('Cost $& more!')
  })

  it('uses name and tagline for the home title unless an SEO title is set', () => {
    expect(homeTitle(site)).toBe('Seafarers Crew | Crewing')
    expect(homeTitle(site, 'Custom')).toBe('Custom')
  })

  it('builds canonical URLs without duplicate or trailing slashes', () => {
    expect(canonicalUrl('https://a.com/', '/about/')).toBe('https://a.com/about')
    expect(canonicalUrl('https://a.com', 'news')).toBe('https://a.com/news')
  })

  it('prefers the page image and falls back to the default OG image', () => {
    expect(resolveOgImage(image('https://x/page'), site.seo.default_og_image)?.url).toBe('https://x/page-1200')
    expect(resolveOgImage(null, site.seo.default_og_image)?.url).toBe('https://x/default-1200')
    expect(resolveOgImage(undefined, null)).toBeNull()
  })

  it('builds complete metadata including twitter and open graph', () => {
    const meta = buildMetadata({ site, baseUrl: 'https://a.com', path: '/about', title: 'About | Seafarers Crew' })
    expect(meta.alternates?.canonical).toBe('https://a.com/about')
    expect(meta.description).toBe('Default description')
    expect(meta.openGraph).toMatchObject({ url: 'https://a.com/about', siteName: 'Seafarers Crew', type: 'website' })
    expect(meta.twitter).toMatchObject({ card: 'summary_large_image' })
  })

  it('marks articles with the article type and dates', () => {
    const meta = buildMetadata({ site, baseUrl: 'https://a.com', path: '/news/x', title: 'X', type: 'article', publishedTime: '2026-01-01T00:00:00Z' })
    expect(meta.openGraph).toMatchObject({ type: 'article', publishedTime: '2026-01-01T00:00:00Z' })
  })
})

describe('json-ld builders', () => {
  it('builds an organization node with logo and sameAs', () => {
    const node = organizationJsonLd(site, 'https://a.com')
    expect(node).toMatchObject({ '@type': 'Organization', name: 'Seafarers Crew', url: 'https://a.com', logo: 'https://x/logo', sameAs: ['https://l'] })
  })

  it('builds a news article node', () => {
    const article = { title: 'T', slug: 't', excerpt: 'E', published_at: 'p', updated_at: 'u', meta_description: '', thumbnail: image('https://x/t') } as unknown as NewsArticle
    expect(newsArticleJsonLd(article, site, 'https://a.com')).toMatchObject({ '@type': 'NewsArticle', headline: 'T', url: 'https://a.com/news/t', image: ['https://x/t'] })
  })

  it('numbers breadcrumb items from 1', () => {
    const node = breadcrumbJsonLd([{ name: 'Home', path: '/' }, { name: 'News', path: '/news' }], 'https://a.com')
    expect(node.itemListElement).toEqual([
      { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://a.com/' },
      { '@type': 'ListItem', position: 2, name: 'News', item: 'https://a.com/news' },
    ])
  })

  it('escapes angle brackets so JSON-LD cannot close the script tag', () => {
    expect(serializeJsonLd({ a: '</script>' })).not.toContain('<')
  })
})
