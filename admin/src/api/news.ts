import { api, type Query } from './client'
import type { AdminImage } from './cmsMedia'

export type NewsStatus = 'draft' | 'published'

export interface NewsSummary {
  id: string
  title: string
  slug: string
  excerpt: string
  thumbnail_file_id: string | null
  thumbnail: AdminImage | null
  published_at: string | null
  status: NewsStatus
  is_live: boolean
  created_at: string
  updated_at: string
}

export interface NewsDetail extends NewsSummary {
  body_html: string
  seo_title: string
  meta_description: string
}

export interface NewsInput {
  title: string
  slug: string
  excerpt: string
  body_html: string
  thumbnail_file_id: string | null
  published_at: string | null
  status: NewsStatus
  seo_title: string
  meta_description: string
}

export const newsApi = {
  list: (query: Query, signal?: AbortSignal) => api.list<NewsSummary[]>('/news', query, signal),
  get: (id: string) => api.get<NewsDetail>(`/news/${id}`),
  create: (input: NewsInput) => api.post<NewsDetail>('/news', input),
  update: (id: string, input: NewsInput) => api.put<NewsDetail>(`/news/${id}`, input),
  remove: (id: string) => api.delete(`/news/${id}`),
}
