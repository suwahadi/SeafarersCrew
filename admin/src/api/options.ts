import { api } from './client'

interface MasterBase {
  id: string
  is_active: boolean
  sort_order: number
}

export interface CompanyOption extends MasterBase {
  code: string
  name: string
  type: 'company' | 'principal' | 'both'
  country: string
}

export interface VesselOption extends MasterBase {
  name: string
  imo_number: string
  flag: string
  vessel_type_id: string | null
  grt: string | null
  hp_kw: string
  principal_id: string | null
  owner_company_id: string | null
  status: string
}

export interface RankOption extends MasterBase {
  name: string
  short_code: string
  department: 'deck' | 'engine' | 'catering_other'
  is_officer: boolean
}

export interface NamedOption extends MasterBase {
  name: string
}

export interface CurrencyOption extends MasterBase {
  code: string
  name: string
  symbol: string
  decimals: number
}

export interface CrewStatusOption extends MasterBase {
  code: string | null
  name: string
  color: string
  is_system: boolean
}

export interface AllowanceTypeOption extends MasterBase {
  name: string
  basis: string
}

export interface DocumentTypeOption extends MasterBase {
  code: string | null
  name: string
  has_expiry: boolean
  number_hint: string
}

export interface CertificateTypeOption extends MasterBase {
  code: string | null
  name: string
  stcw_reference: string
  category: string
  has_expiry: boolean
}

export interface MasterOptions {
  companies: CompanyOption[]
  vessels: VesselOption[]
  ranks: RankOption[]
  vessel_types: NamedOption[]
  currencies: CurrencyOption[]
  crew_statuses: CrewStatusOption[]
  allowance_types: AllowanceTypeOption[]
  document_types: DocumentTypeOption[]
  certificate_types: CertificateTypeOption[]
}

export const optionsApi = {
  all: (signal?: AbortSignal) => api.get<MasterOptions>('/options', undefined, signal),
}

export const OPTIONS_KEY = ['options'] as const

export const DEPARTMENTS = [
  { value: 'deck', label: 'Deck' },
  { value: 'engine', label: 'Engine' },
  { value: 'catering_other', label: 'Catering/Other' },
] as const

export function departmentLabel(value: string): string {
  return DEPARTMENTS.find((d) => d.value === value)?.label ?? ''
}
