import type { MetadataRoute } from 'next'

export function buildRobots(indexingEnabled: boolean, baseUrl: string): MetadataRoute.Robots {
  return {
    rules: indexingEnabled
      ? { userAgent: '*', allow: '/', disallow: '/api/' }
      : { userAgent: '*', disallow: '/' },
    sitemap: `${baseUrl}/sitemap.xml`,
  }
}
