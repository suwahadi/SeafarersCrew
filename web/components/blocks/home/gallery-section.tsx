import Link from 'next/link'
import { Container } from '@/components/ui/container'
import { LightboxGallery } from '@/components/ui/lightbox-gallery'
import { SectionHeading } from '@/components/ui/section-heading'
import type { GalleryItem } from '@/lib/types'

export function GallerySection({ items }: { items: readonly GalleryItem[] }) {
  return (
    <section aria-labelledby="home-gallery" className="py-16 sm:py-20">
      <Container>
        <SectionHeading
          id="home-gallery"
          eyebrow="Gallery"
          title="Life on board"
          action={
            <Link href="/gallery" className="text-sm font-semibold text-accent-text hover:underline">
              View full gallery
            </Link>
          }
        />
        <LightboxGallery items={items} columns={4} lightbox />
      </Container>
    </section>
  )
}
