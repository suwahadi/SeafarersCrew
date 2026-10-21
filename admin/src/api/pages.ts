import { api, type Query } from './client'
import type { AdminImage } from './cmsMedia'

export type PageStatus = 'draft' | 'published'
export type BlockType = 'rich_text' | 'image_text' | 'gallery' | 'video' | 'cta' | 'stats' | 'faq' | 'card_grid'

export interface PageSummary {
  id: string
  title: string
  slug: string
  path: string
  system_key: string | null
  is_system: boolean
  status: PageStatus
  in_header_menu: boolean
  in_footer_menu: boolean
  menu_order: number
  published_at: string | null
  created_at: string
  updated_at: string
}

export interface StatItem {
  number: number
  suffix: string
  label: string
}

export interface FaqItem {
  question: string
  answer_html: string
}

export interface Card {
  icon?: string
  image_file_id?: string
  title: string
  text: string
  link?: string
}

interface BlockBase<T extends BlockType, S> {
  id: string
  type: T
  hidden: boolean
  settings: S
}

export type RichTextBlock = BlockBase<'rich_text', { html: string }>
export type ImageTextBlock = BlockBase<
  'image_text',
  { image_file_id: string; heading: string; text_html: string; button_label: string; button_link: string; image_position: 'left' | 'right' }
>
export type GalleryBlock = BlockBase<'gallery', { image_ids: string[]; columns: number; lightbox: boolean }>
export type VideoBlock = BlockBase<'video', { url: string; title: string; provider?: string; video_id?: string }>
export type CtaBlock = BlockBase<
  'cta',
  { heading: string; text: string; button_label: string; button_link: string; background_image_file_id?: string; background_color?: string }
>
export type StatsBlock = BlockBase<'stats', { items: StatItem[] }>
export type FaqBlock = BlockBase<'faq', { items: FaqItem[] }>
export type CardGridBlock = BlockBase<'card_grid', { columns: number; cards: Card[] }>

export type PageBlock = RichTextBlock | ImageTextBlock | GalleryBlock | VideoBlock | CtaBlock | StatsBlock | FaqBlock | CardGridBlock

export type PageMedia = Record<string, AdminImage>

export interface PageDetail extends PageSummary {
  intro_html: string
  seo_title: string
  meta_description: string
  og_image_file_id: string | null
  og_image: AdminImage | null
  blocks: PageBlock[]
  media: PageMedia
}

export interface PageInput {
  title: string
  slug: string
  seo_title: string
  meta_description: string
  og_image_file_id: string | null
  status: PageStatus
  in_header_menu: boolean
  in_footer_menu: boolean
  menu_order: number
  intro_html: string
  blocks: PageBlock[]
}

export const pagesApi = {
  list: (query: Query, signal?: AbortSignal) => api.list<PageSummary[]>('/pages', query, signal),
  get: (id: string) => api.get<PageDetail>(`/pages/${id}`),
  create: (input: PageInput) => api.post<PageDetail>('/pages', input),
  update: (id: string, input: PageInput) => api.put<PageDetail>(`/pages/${id}`, input),
  remove: (id: string) => api.delete(`/pages/${id}`),
}
