import type { Query } from '@/api/client'

export interface PayrollFilters {
  bank: string[]
  vessel: string[]
  rank: string[]
  has_remark: boolean
  no_usd: boolean
}

export const EMPTY_FILTERS: PayrollFilters = { bank: [], vessel: [], rank: [], has_remark: false, no_usd: false }

const LIST_KEYS = ['bank', 'vessel', 'rank'] as const
const FLAG_KEYS = ['has_remark', 'no_usd'] as const

export function filtersFromParams(params: URLSearchParams): PayrollFilters {
  return {
    bank: params.getAll('bank'),
    vessel: params.getAll('vessel'),
    rank: params.getAll('rank'),
    has_remark: params.get('has_remark') === '1',
    no_usd: params.get('no_usd') === '1',
  }
}

export function filtersToParams(f: PayrollFilters): URLSearchParams {
  const params = new URLSearchParams()
  for (const key of LIST_KEYS) {
    for (const v of f[key]) params.append(key, v)
  }
  for (const key of FLAG_KEYS) {
    if (f[key]) params.set(key, '1')
  }
  return params
}

export function filtersToQuery(f: PayrollFilters): Query {
  return { bank: f.bank, vessel: f.vessel, rank: f.rank, has_remark: f.has_remark ? '1' : '', no_usd: f.no_usd ? '1' : '' }
}

export function toSearchParams(query: Query): URLSearchParams {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue
    if (Array.isArray(value)) {
      for (const v of value) params.append(key, String(v))
    } else {
      params.set(key, String(value))
    }
  }
  return params
}

export function activeFilterCount(f: PayrollFilters): number {
  return LIST_KEYS.filter((k) => f[k].length > 0).length + FLAG_KEYS.filter((k) => f[k]).length
}
