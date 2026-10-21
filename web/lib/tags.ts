export const CACHE_TAGS = ['site', 'home', 'pages', 'news', 'gallery', 'vacancies', 'redirects', 'sitemap'] as const

export type CacheTag = (typeof CACHE_TAGS)[number]

export function isCacheTag(value: unknown): value is CacheTag {
  return typeof value === 'string' && (CACHE_TAGS as readonly string[]).includes(value)
}
