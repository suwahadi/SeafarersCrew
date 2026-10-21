import type { MetadataRoute } from 'next'
import { list } from '@/lib/list'
import { canonicalUrl } from '@/lib/seo'
import type { SitemapData, SitemapEntry } from '@/lib/types'

export function buildSitemapEntries(data: SitemapData, baseUrl: string): MetadataRoute.Sitemap {
  const groups: (readonly SitemapEntry[])[] = [list(data.system_pages), list(data.pages), list(data.news), list(data.vacancies)]
  const byUrl = new Map<string, MetadataRoute.Sitemap[number]>()
  byUrl.set(canonicalUrl(baseUrl, '/'), { url: canonicalUrl(baseUrl, '/') })

  for (const entry of groups.flat()) {
    const url = canonicalUrl(baseUrl, entry.path)
    const lastModified = entry.lastmod ? new Date(entry.lastmod) : undefined
    byUrl.set(url, {
      url,
      lastModified: lastModified && !Number.isNaN(lastModified.getTime()) ? lastModified : undefined,
    })
  }
  return [...byUrl.values()]
}
