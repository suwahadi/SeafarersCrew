import { queryOptions } from '@tanstack/react-query'
import { api, type Query } from './client'

export interface MasterBase {
  id: string
  is_active: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface Company extends MasterBase {
  code: string
  name: string
  type: 'company' | 'principal' | 'both'
  address: string
  country: string
  phone: string
  email: string
  website: string
  logo_file_id: string | null
}

export interface Vessel extends MasterBase {
  name: string
  imo_number: string
  flag: string
  vessel_type_id: string | null
  grt: string | null
  hp_kw: string
  year_built: number | null
  owner_company_id: string | null
  principal_id: string | null
  status: 'active' | 'laid_up' | 'sold'
}

export interface Rank extends MasterBase {
  name: string
  short_code: string
  department: 'deck' | 'engine' | 'catering_other'
  is_officer: boolean
}

export interface VesselType extends MasterBase {
  name: string
}

export interface Currency extends MasterBase {
  code: string
  name: string
  symbol: string
  decimals: number
}

export interface CrewStatus extends MasterBase {
  code: string | null
  name: string
  color: string
  is_system: boolean
}

export interface AllowanceType extends MasterBase {
  name: string
  basis: 'fixed_month' | 'fixed_day' | 'percent_basic'
}

export interface DocumentType extends MasterBase {
  code: string | null
  name: string
  has_expiry: boolean
  number_hint: string
}

export interface CertificateType extends MasterBase {
  code: string | null
  name: string
  stcw_reference: string
  category: 'coc' | 'cop' | 'endorsement' | 'other'
  has_expiry: boolean
}

export interface MasterOptions {
  companies: Company[]
  vessels: Vessel[]
  ranks: Rank[]
  vessel_types: VesselType[]
  currencies: Currency[]
  crew_statuses: CrewStatus[]
  allowance_types: AllowanceType[]
  document_types: DocumentType[]
  certificate_types: CertificateType[]
}

export type MasterSegment =
  | 'companies'
  | 'vessels'
  | 'ranks'
  | 'vessel-types'
  | 'currencies'
  | 'crew-statuses'
  | 'allowance-types'
  | 'document-types'
  | 'certificate-types'

export type MasterRecord = MasterBase & Record<string, unknown>

export const OPTIONS_KEY = ['options'] as const

export const optionsQuery = queryOptions({
  queryKey: OPTIONS_KEY,
  queryFn: ({ signal }) => api.get<MasterOptions>('/options', undefined, signal),
  staleTime: 5 * 60_000,
})

export function masterApi(segment: MasterSegment) {
  const base = `/${segment}`
  return {
    list: (query: Query, signal?: AbortSignal) => api.list<MasterRecord[]>(base, query, signal),
    get: (id: string) => api.get<MasterRecord>(`${base}/${id}`),
    create: (body: Record<string, unknown>) => api.post<MasterRecord>(base, body),
    update: (id: string, body: Record<string, unknown>) => api.put<MasterRecord>(`${base}/${id}`, body),
    remove: (id: string) => api.delete(`${base}/${id}`),
    reorder: (ids: string[]) => api.patch<undefined>(`${base}/reorder`, { ids }),
  }
}

export async function fetchAllMasterRows(segment: MasterSegment): Promise<MasterRecord[]> {
  const out: MasterRecord[] = []
  for (let page = 1; ; page++) {
    const res = await masterApi(segment).list({ page, per_page: 100, sort: 'sort_order', order: 'asc' })
    out.push(...res.data)
    if (out.length >= res.meta.total || res.data.length === 0) return out
  }
}

export interface ComplianceCell {
  rank_id: string
  document_type_id: string | null
  certificate_type_id: string | null
}

export interface ComplianceMatrix {
  ranks: Rank[]
  document_types: DocumentType[]
  certificate_types: CertificateType[]
  cells: ComplianceCell[]
}

export const COMPLIANCE_KEY = ['compliance-matrix'] as const

export const complianceApi = {
  get: (signal?: AbortSignal) => api.get<ComplianceMatrix>('/compliance-matrix', undefined, signal),
  replace: (cells: ComplianceCell[]) => api.put<ComplianceMatrix>('/compliance-matrix', { cells }),
}
