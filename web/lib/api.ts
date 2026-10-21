import 'server-only'
import { notFound } from 'next/navigation'
import { apiUrl } from '@/lib/env'
import type {
  CardGridSettings,
  CmsPage,
  Home,
  NewsArticle,
  NewsCard,
  Paged,
  RedirectRule,
  Site,
  SitemapData,
  VacancyCard,
  GalleryItem,
} from '@/lib/types'
import { findBlock } from '@/lib/blocks'
import type { CacheTag } from '@/lib/tags'

const REVALIDATE_SECONDS = 60
const REQUEST_TIMEOUT_MS = 10_000
const BASE_PATH = '/api/public/v1'

export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

interface Envelope<T> {
  data: T
  meta?: Paged<unknown>['meta']
}

async function request<T>(path: string, tag: CacheTag): Promise<Envelope<T> | null> {
  const response = await fetch(`${apiUrl()}${BASE_PATH}${path}`, {
    headers: { accept: 'application/json' },
    signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
    next: { tags: [tag], revalidate: REVALIDATE_SECONDS },
  })
  if (response.status === 404) {
    return null
  }
  if (!response.ok) {
    throw new ApiError(response.status, `Public API ${path} responded with ${String(response.status)}`)
  }
  return (await response.json()) as Envelope<T>
}

async function requireData<T>(path: string, tag: CacheTag): Promise<T> {
  const body = await request<T>(path, tag)
  if (!body) {
    notFound()
  }
  return body.data
}

async function optionalData<T>(path: string, tag: CacheTag): Promise<T | null> {
  const body = await request<T>(path, tag)
  return body ? body.data : null
}

function encodePath(path: string): string {
  return path
    .split('/')
    .filter((segment) => segment !== '')
    .map(encodeURIComponent)
    .join('/')
}

function queryString(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') {
      search.set(key, String(value))
    }
  }
  const text = search.toString()
  return text === '' ? '' : `?${text}`
}

export function getSite(): Promise<Site> {
  return requireData<Site>('/site', 'site')
}

export function getHome(): Promise<Home> {
  return requireData<Home>('/home', 'home')
}

export function findPage(slug: string): Promise<CmsPage | null> {
  return optionalData<CmsPage>(`/pages/${encodeURIComponent(slug)}`, 'pages')
}

export interface ServicesGrid {
  title: string
  settings: CardGridSettings
}

export async function getServicesGrid(): Promise<ServicesGrid | null> {
  const page = await findPage('services')
  const settings = page ? findBlock(page.blocks, 'card_grid')?.settings : undefined
  return page && settings ? { title: page.title, settings } : null
}

export async function getNewsList(page: number, perPage: number): Promise<Paged<NewsCard>> {
  const body = await request<NewsCard[]>(`/news${queryString({ page, per_page: perPage })}`, 'news')
  if (!body?.meta) {
    throw new ApiError(502, 'Public API /news returned no pagination meta')
  }
  return { data: body.data, meta: body.meta }
}

export function findArticle(slug: string): Promise<NewsArticle | null> {
  return optionalData<NewsArticle>(`/news/${encodeURIComponent(slug)}`, 'news')
}

export function getGallery(): Promise<GalleryItem[]> {
  return requireData<GalleryItem[]>('/gallery', 'gallery')
}

export interface VacancyQuery {
  q?: string
  rank?: string
  vessel_type?: string
  department?: string
  location?: string
  page?: number
  per_page?: number
}

export async function getVacancies(query: VacancyQuery = {}): Promise<Paged<VacancyCard>> {
  const body = await request<VacancyCard[]>(`/vacancies${queryString({ ...query })}`, 'vacancies')
  if (!body?.meta) {
    throw new ApiError(502, 'Public API /vacancies returned no pagination meta')
  }
  return { data: body.data, meta: body.meta }
}

export interface VacancyDetail extends VacancyCard {
  duration: string
  salary_text: string
  description_html: string
  requirements_html: string
  required_documents: string[]
  is_open: boolean
  updated_at: string
}

export function findVacancy(slug: string): Promise<VacancyDetail | null> {
  return optionalData<VacancyDetail>(`/vacancies/${encodeURIComponent(slug)}`, 'vacancies')
}

export interface VacancyFilters {
  ranks: { id: string; name: string; short_code: string }[]
  departments: string[]
  vessel_types: { id: string; name: string }[]
  locations: string[]
}

export function getVacancyFilters(): Promise<VacancyFilters> {
  return requireData<VacancyFilters>('/vacancy-filters', 'vacancies')
}

export function findRedirect(path: string): Promise<RedirectRule | null> {
  return optionalData<RedirectRule>(`/redirects/${encodePath(path)}`, 'redirects')
}

export function getSitemapData(): Promise<SitemapData> {
  return requireData<SitemapData>('/sitemap-data', 'sitemap')
}
