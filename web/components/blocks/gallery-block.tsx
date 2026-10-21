import { LightboxGallery } from '@/components/ui/lightbox-gallery'
import { list } from '@/lib/list'
import type { GallerySettings } from '@/lib/types'

export function GalleryBlock({ settings }: { settings: GallerySettings }) {
  const images = list(settings.images)
  if (images.length === 0) {
    return null
  }
  return <LightboxGallery items={images} columns={settings.columns} lightbox={settings.lightbox} />
}
