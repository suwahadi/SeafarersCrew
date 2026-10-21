import type { Query } from '@/api/client'

export interface CrewFilters {
  status: string[]
  rank_id: string[]
  department: string
  vessel_type_id: string
  vessel_id: string
  principal_id: string
  age_min: string
  age_max: string
  nationality: string
  compliance: string
  doc_validity: string
  doc_days: string
  sea_rank_id: string
  sea_min_years: string
  sea_min_months: string
  group_by: string
}

export const EMPTY_FILTERS: CrewFilters = {
  status: [],
  rank_id: [],
  department: '',
  vessel_type_id: '',
  vessel_id: '',
  principal_id: '',
  age_min: '',
  age_max: '',
  nationality: '',
  compliance: '',
  doc_validity: '',
  doc_days: '',
  sea_rank_id: '',
  sea_min_years: '',
  sea_min_months: '',
  group_by: '',
}

const LIST_KEYS = ['status', 'rank_id'] as const
const SCALAR_KEYS = [
  'department',
  'vessel_type_id',
  'vessel_id',
  'principal_id',
  'age_min',
  'age_max',
  'nationality',
  'compliance',
  'doc_validity',
  'doc_days',
  'sea_rank_id',
  'sea_min_years',
  'sea_min_months',
  'group_by',
] as const

export function filtersFromParams(params: URLSearchParams): CrewFilters {
  const f: CrewFilters = { ...EMPTY_FILTERS, status: [], rank_id: [] }
  for (const key of LIST_KEYS) {
    f[key] = (params.get(key) ?? '').split(',').filter(Boolean)
  }
  for (const key of SCALAR_KEYS) {
    f[key] = params.get(key) ?? ''
  }
  return f
}

export function filtersToParams(f: CrewFilters): URLSearchParams {
  const params = new URLSearchParams()
  for (const key of LIST_KEYS) {
    if (f[key].length > 0) params.set(key, f[key].join(','))
  }
  for (const key of SCALAR_KEYS) {
    if (f[key] !== '') params.set(key, f[key])
  }
  return params
}

function toInt(value: string): number | undefined {
  if (value.trim() === '') return undefined
  const n = Number.parseInt(value, 10)
  return Number.isFinite(n) && n >= 0 ? n : undefined
}

export function seaMinDays(years: string, months: string): number | undefined {
  const y = toInt(years) ?? 0
  const m = toInt(months) ?? 0
  const days = y * 365 + m * 30
  return days > 0 ? days : undefined
}

export function filtersToQuery(f: CrewFilters): Query {
  const minDays = f.sea_rank_id ? seaMinDays(f.sea_min_years, f.sea_min_months) : undefined
  return {
    status: f.status,
    rank_id: f.rank_id,
    department: f.department,
    vessel_type_id: f.vessel_type_id,
    vessel_id: f.vessel_id,
    principal_id: f.principal_id,
    age_min: toInt(f.age_min),
    age_max: toInt(f.age_max),
    nationality: f.nationality.trim(),
    compliance: f.compliance,
    doc_validity: f.doc_validity,
    doc_days: f.doc_validity ? toInt(f.doc_days) : undefined,
    sea_rank_id: minDays ? f.sea_rank_id : '',
    sea_min_days: minDays,
    group_by: f.group_by,
  }
}

export function activeFilterCount(f: CrewFilters): number {
  let n = f.status.length > 0 ? 1 : 0
  n += f.rank_id.length > 0 ? 1 : 0
  for (const key of SCALAR_KEYS) {
    if (key === 'group_by' || key === 'doc_days' || key === 'sea_min_years' || key === 'sea_min_months') continue
    if (f[key] !== '') n++
  }
  return n
}
