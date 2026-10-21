import type { Metadata } from 'next'
import { Container } from '@/components/ui/container'
import { LightboxGallery } from '@/components/ui/lightbox-gallery'
import { PageHeader } from '@/components/ui/page-header'
import { findPage, getGallery, getSite } from '@/lib/api'
import { appUrl } from '@/lib/env'
import { buildMetadata, pageTitle } from '@/lib/seo'
import { firstNonEmpty } from '@/lib/text'

const FALLBACK_TITLE = 'Gallery'

export async function generateMetadata(): Promise<Metadata> {
  const [site, page] = await Promise.all([getSite(), findPage('gallery')])
  return buildMetadata({
    site,
    baseUrl: appUrl(),
    path: '/gallery',
    title: pageTitle(site, firstNonEmpty(page?.seo_title, page?.title, FALLBACK_TITLE)),
    description: page?.meta_description,
    image: page?.og_image,
  })
}

export default async function GalleryPage() {
  const [items, page] = await Promise.all([getGallery(), findPage('gallery')])
  const title = firstNonEmpty(page?.title, FALLBACK_TITLE)
  return (
    <>
      <PageHeader
        title={title}
        introHtml={page?.intro_html}
        crumbs={[
          { name: 'Home', path: '/' },
          { name: title, path: '/gallery' },
        ]}
      />
      <Container className="py-12 sm:py-16">
        {items.length === 0 ? (
          <p className="rounded-lg border border-line bg-surface px-6 py-12 text-center text-muted">
            There are no photos yet. Please check back soon.
          </p>
        ) : (
          <LightboxGallery items={items} columns={4} lightbox />
        )}
      </Container>
    </>
  )
}
