import type { Metadata } from 'next'
import { notFound, permanentRedirect } from 'next/navigation'
import { BlockRenderer } from '@/components/blocks/block-renderer'
import { PageHeader } from '@/components/ui/page-header'
import { findPage, getSite } from '@/lib/api'
import { appUrl } from '@/lib/env'
import { notFoundOrRedirect } from '@/lib/redirects'
import { buildMetadata, pageTitle } from '@/lib/seo'
import { firstNonEmpty } from '@/lib/text'

interface PageProps {
  params: Promise<{ slug: string }>
}

const SYSTEM_SLUGS: ReadonlySet<string> = new Set(['news', 'career', 'contact', 'gallery'])

function guardSlug(slug: string): void {
  if (slug === 'home') {
    permanentRedirect('/')
  }
  if (SYSTEM_SLUGS.has(slug) || slug.includes('.')) {
    notFound()
  }
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { slug } = await params
  guardSlug(slug)
  const [site, page] = await Promise.all([getSite(), findPage(slug)])
  if (!page) {
    return { title: 'Page not found', robots: { index: false } }
  }
  return buildMetadata({
    site,
    baseUrl: appUrl(),
    path: `/${page.slug}`,
    title: pageTitle(site, firstNonEmpty(page.seo_title, page.title)),
    description: page.meta_description,
    image: page.og_image,
  })
}

export default async function CmsPage({ params }: PageProps) {
  const { slug } = await params
  guardSlug(slug)
  const page = await findPage(slug)
  if (!page) {
    return notFoundOrRedirect(slug)
  }
  return (
    <>
      <PageHeader
        title={page.title}
        introHtml={page.intro_html}
        crumbs={[
          { name: 'Home', path: '/' },
          { name: page.title, path: `/${page.slug}` },
        ]}
      />
      <BlockRenderer blocks={page.blocks} />
    </>
  )
}
