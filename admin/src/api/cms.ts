import { api, type Query } from './client'
import type { AdminImage } from './cmsMedia'
import type { BulkRequest } from '@/lib/bulkUpload'
import type { MenuPayloadItem, ServerMenuItem } from '@/lib/menuTree'

export interface CmsBase {
  id: string
  is_active: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface HeroSlide extends CmsBase {
  image: AdminImage | null
  mobile_image: AdminImage | null
  heading: string
  subheading: string
  button_label: string
  button_link: string
}

export interface GalleryItem extends CmsBase {
  image: AdminImage | null
  caption: string
  alt_text: string
  show_on_home: boolean
}

export interface Testimonial extends CmsBase {
  name: string
  position: string
  photo: AdminImage | null
  quote: string
  rating: number
}

export interface ClientLogo extends CmsBase {
  name: string
  logo: AdminImage | null
  url: string
}

export interface Stat extends CmsBase {
  number: number
  suffix: string
  label: string
}

export interface Language extends CmsBase {
  name: string
  code: string
  flag: AdminImage | null
  flag_code: string | null
  is_source: boolean
}

export type CmsSegment = 'hero-slides' | 'gallery' | 'testimonials' | 'client-logos' | 'stats' | 'languages'

export function cmsApi<T extends CmsBase>(segment: CmsSegment) {
  const base = `/${segment}`
  return {
    list: (query: Query, signal?: AbortSignal) => api.list<T[]>(base, query, signal),
    create: (body: Record<string, unknown>) => api.post<T>(base, body),
    update: (id: string, body: Record<string, unknown>) => api.put<T>(`${base}/${id}`, body),
    remove: (id: string) => api.delete(`${base}/${id}`),
    reorder: (ids: string[]) => api.patch<undefined>(`${base}/reorder`, { ids }),
  }
}

export const cmsKey = (segment: CmsSegment) => ['cms', segment] as const

export async function fetchAllCms<T extends CmsBase>(segment: CmsSegment, signal?: AbortSignal): Promise<T[]> {
  const out: T[] = []
  for (let page = 1; ; page++) {
    const res = await cmsApi<T>(segment).list({ page, per_page: 100, sort: 'sort_order', order: 'asc' }, signal)
    out.push(...res.data)
    if (out.length >= res.meta.total || res.data.length === 0) return out
  }
}

export const galleryApi = {
  bulk: (request: BulkRequest) => api.post<GalleryItem[]>('/gallery/bulk', request),
}

export type MenuName = 'header' | 'footer'

export interface MenusResponse {
  header: ServerMenuItem[]
  footer: ServerMenuItem[]
}

export const MENUS_KEY = ['menus'] as const

export const menusApi = {
  get: (signal?: AbortSignal) => api.get<MenusResponse>('/menus', undefined, signal),
  replace: (menu: MenuName, items: MenuPayloadItem[]) => api.put<MenusResponse>(`/menus/${menu}`, { items }),
}

export interface PageSummary {
  id: string
  title: string
  slug: string
  path: string
  system_key: string | null
  is_system: boolean
  status: 'draft' | 'published'
}

export const PAGE_OPTIONS_KEY = ['pages', 'menu-options'] as const

export async function fetchAllPages(signal?: AbortSignal): Promise<PageSummary[]> {
  const out: PageSummary[] = []
  for (let page = 1; ; page++) {
    const res = await api.list<PageSummary[]>('/pages', { page, per_page: 100 }, signal)
    out.push(...res.data)
    if (out.length >= res.meta.total || res.data.length === 0) return out
  }
}
