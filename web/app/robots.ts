import type { MetadataRoute } from 'next'
import { connection } from 'next/server'
import { getSite } from '@/lib/api'
import { appUrl } from '@/lib/env'
import { buildRobots } from '@/lib/robots'

export default async function robots(): Promise<MetadataRoute.Robots> {
  await connection()
  const site = await getSite()
  return buildRobots(site.seo.robots_indexing, appUrl())
}
