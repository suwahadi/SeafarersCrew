import type { MetadataRoute } from 'next'
import { connection } from 'next/server'
import { getSitemapData } from '@/lib/api'
import { appUrl } from '@/lib/env'
import { buildSitemapEntries } from '@/lib/sitemap'

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  await connection()
  const data = await getSitemapData()
  return buildSitemapEntries(data, appUrl())
}
