import type { Metadata } from 'next'
import { HomeSections } from '@/components/blocks/home/home-sections'
import { getHome, getServicesGrid, getSite } from '@/lib/api'
import { appUrl } from '@/lib/env'
import { buildMetadata, homeTitle } from '@/lib/seo'

export async function generateMetadata(): Promise<Metadata> {
  const [site, home] = await Promise.all([getSite(), getHome()])
  return buildMetadata({
    site,
    baseUrl: appUrl(),
    path: '/',
    title: homeTitle(site, home.page?.seo_title),
    description: home.page?.meta_description,
    image: home.page?.og_image,
  })
}

export default async function HomePage() {
  const [site, home, services] = await Promise.all([getSite(), getHome(), getServicesGrid()])
  return <HomeSections home={home} siteName={site.name} services={services} />
}
