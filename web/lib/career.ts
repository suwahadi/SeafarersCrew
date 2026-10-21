import type { VacancyQuery } from '@/lib/api'

export const CAREER_PER_PAGE = 12
export const DEPARTMENTS = ['deck', 'engine', 'catering_other'] as const

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const MAX_TEXT_LENGTH = 100
const FILTER_KEYS = ['q', 'rank', 'vessel_type', 'department', 'location'] as const

type RawParams = Record<string, string | string[] | undefined>

export interface CareerFilters {
  q: string
  rank: string
  vessel_type: string
  department: string
  location: string
}

export interface CareerQuery extends CareerFilters {
  page: number
}

function single(raw: string | string[] | undefined): string {
  return (Array.isArray(raw) ? raw[0] : raw) ?? ''
}

function cleanText(raw: string | string[] | undefined): string {
  return single(raw).replace(/\s+/g, ' ').trim().slice(0, MAX_TEXT_LENGTH)
}

function cleanUuid(raw: string | string[] | undefined): string {
  const value = single(raw).trim()
  return UUID_PATTERN.test(value) ? value.toLowerCase() : ''
}

function cleanDepartment(raw: string | string[] | undefined): string {
  const value = single(raw).trim()
  return (DEPARTMENTS as readonly string[]).includes(value) ? value : ''
}

export function parseCareerQuery(params: RawParams): CareerQuery {
  const page = Number.parseInt(single(params.page), 10)
  return {
    q: cleanText(params.q),
    rank: cleanUuid(params.rank),
    vessel_type: cleanUuid(params.vessel_type),
    department: cleanDepartment(params.department),
    location: cleanText(params.location),
    page: Number.isFinite(page) && page >= 1 ? page : 1,
  }
}

export function hasActiveFilters(filters: CareerFilters): boolean {
  return FILTER_KEYS.some((key) => filters[key] !== '')
}

export function toVacancyQuery(query: CareerQuery): VacancyQuery {
  return {
    q: query.q,
    rank: query.rank,
    vessel_type: query.vessel_type,
    department: query.department,
    location: query.location,
    page: query.page,
    per_page: CAREER_PER_PAGE,
  }
}

export function careerHref(filters: CareerFilters, page: number, basePath = '/career'): string {
  const search = new URLSearchParams()
  for (const key of FILTER_KEYS) {
    if (filters[key] !== '') {
      search.set(key, filters[key])
    }
  }
  if (page > 1) {
    search.set('page', String(page))
  }
  const text = search.toString()
  return text === '' ? basePath : `${basePath}?${text}`
}
