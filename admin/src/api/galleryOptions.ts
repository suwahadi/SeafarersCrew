import { api } from './client'
import type { AdminImage } from './cmsMedia'

export interface GalleryOption {
  id: string
  caption: string
  alt_text: string
  image: AdminImage | null
}

export const galleryOptionsApi = {
  active: () => api.list<GalleryOption[]>('/gallery', { per_page: 100, is_active: true }).then((r) => r.data),
}
