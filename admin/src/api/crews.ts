import type { Span } from '@/lib/format'
import { api, buildQuery, type Job, type Query } from './client'

export interface CrewStatusBadge {
  code: string
  name: string
  color: string
}

export type Gender = 'male' | 'female'
export type MaritalStatus = 'single' | 'married' | 'divorced' | 'widowed'

export interface Crew {
  id: string
  full_name: string
  place_of_birth: string
  date_of_birth: string | null
  age: number | null
  gender: Gender | null
  nationality: string
  religion: string
  marital_status: MaritalStatus | null
  height_cm: number | null
  weight_kg: number | null
  shoe_size: string
  coverall_size: string
  applied_rank_id: string | null
  applied_rank_name: string
  status: CrewStatusBadge
  status_reason: string
  photo_file_id: string | null
  cv_file_id: string | null
  address: string
  city: string
  province: string
  country: string
  postal_code: string
  home_phone: string
  mobile_phone: string
  email: string
  nearest_airport: string
  created_at: string
  updated_at: string
}

export interface CrewInput {
  full_name: string
  place_of_birth: string
  date_of_birth: string
  gender: string
  nationality: string
  religion: string
  marital_status: string
  height_cm: number | null
  weight_kg: number | null
  shoe_size: string
  coverall_size: string
  applied_rank_id: string
  photo_file_id: string
  cv_file_id: string
  address: string
  city: string
  province: string
  country: string
  postal_code: string
  home_phone: string
  mobile_phone: string
  email: string
  nearest_airport: string
}

export interface CrewRow {
  id: string
  full_name: string
  date_of_birth: string | null
  age: number | null
  nationality: string
  mobile_phone: string
  email: string
  status: CrewStatusBadge
  rank_id: string | null
  rank_name: string
  department: string
  contract_id: string
  vessel_id: string | null
  vessel_name: string
  vessel_type: string
  principal_id: string | null
  principal_name: string
  employer_name: string
  passport_number: string
  seaman_book_number: string
  compliant: boolean
  nearest_expiry: string | null
  expiry_badge: string
  sea_service: Span
  group_key?: string
}

export interface CrewGroup {
  key: string
  name: string
  count: number
}

export interface CrewList {
  items: CrewRow[]
  groups?: CrewGroup[]
}

export interface BankAccount {
  id: string
  bank_name: string
  account_name: string
  account_number: string
  currency_code: string
  cover_bank: string
  remarks: string
  is_primary: boolean
}

export type BankAccountInput = Omit<BankAccount, 'id'>

export type Relationship = 'spouse' | 'parent' | 'child' | 'sibling' | 'other'

export interface NextOfKin {
  id: string
  name: string
  relationship: Relationship
  phone: string
  address: string
  is_primary: boolean
}

export type NextOfKinInput = Omit<NextOfKin, 'id'>

export interface CrewDocument {
  id: string
  type_id: string
  type_name: string
  has_expiry: boolean
  number: string
  issued_date: string | null
  expired_date: string | null
  issuing_authority?: string
  place_of_issue: string
  file_id: string | null
  badge: string
  days_left: number | null
}

export interface CrewDocumentInput {
  type_id: string
  number: string
  issued_date: string
  expired_date: string
  issuing_authority: string
  place_of_issue: string
  file_id: string
}

export type DocumentKind = 'documents' | 'certificates'

export interface Experience {
  id: string
  vessel_name: string
  vessel_id: string | null
  rank_id: string
  rank_name: string
  vessel_type_id: string | null
  vessel_type: string
  grt_hp: string
  company_id: string | null
  company_name: string
  principal_id: string | null
  principal_name: string
  sign_on_date: string
  sign_off_date: string | null
  status: string
  remarks: string
  source: 'manual' | 'contract'
  contract_id: string | null
  days?: number
  duration?: Span
}

export interface ExperienceInput {
  vessel_name: string
  vessel_id: string
  rank_id: string
  vessel_type_id: string
  grt_hp: string
  company_id: string
  company_name: string
  principal_id: string
  principal_name: string
  sign_on_date: string
  sign_off_date: string
  remarks: string
}

export interface RankSeaService {
  rank_id: string
  rank_name: string
  total: Span
}

export interface SeaService {
  rows: Experience[]
  by_rank: RankSeaService[]
  total: Span
}

export interface ComplianceItem {
  kind: 'document' | 'certificate'
  type_id: string
  type_name: string
  status: 'valid' | 'expiring' | 'expired' | 'missing'
  badge: string
  number?: string
  expired_date: string | null
  days_left: number | null
}

export interface Compliance {
  rank_id: string | null
  check_date: string
  compliant: boolean
  items: ComplianceItem[]
}

export interface Ref {
  id: string
  name: string
}

export interface Assignment {
  contract_id: string
  vessel: Ref
  vessel_type: string
  rank: Ref
  principal: Ref
  sign_on_date: string
  planned_sign_off_date: string
  planned_relief_date: string
  days_on_board: number
}

export type StatusAction = 'medical_leave' | 'ready' | 'blacklist'

export interface ImportPreviewRow {
  line: number
  status: 'valid' | 'invalid' | 'duplicate'
  values: Record<string, string>
  errors?: Record<string, string>
}

export interface ImportPreview {
  token: string
  total: number
  valid: number
  invalid: number
  duplicate: number
  rows: ImportPreviewRow[]
}

export interface ImportOutcome {
  line: number
  crew_id?: string
  error?: string
}

export interface ImportReport {
  created: number
  skipped: number
  failed: number
  outcomes: ImportOutcome[]
}

export interface ExportResult {
  file_id: string
  filename: string
  rows: number
}

const base = (id: string) => `/crews/${encodeURIComponent(id)}`

export const crewsApi = {
  list: (query: Query, signal?: AbortSignal) => api.list<CrewList>('/crews', query, signal),
  get: (id: string) => api.get<Crew>(base(id)),
  create: (input: CrewInput) => api.post<Crew>('/crews', input),
  update: (id: string, input: CrewInput) => api.put<Crew>(base(id), input),
  remove: (id: string) => api.delete(base(id)),
  setStatus: (id: string, action: StatusAction, reason: string) => api.patch<Crew>(`${base(id)}/status`, { action, reason }),
  compliance: (id: string, query: { rank_id?: string; date?: string }) => api.get<Compliance>(`${base(id)}/compliance`, query),
  seaService: (id: string) => api.get<SeaService>(`${base(id)}/sea-service`),
  currentAssignment: (id: string) => api.get<Assignment | null>(`${base(id)}/current-assignment`),

  bankAccounts: (id: string) => api.get<BankAccount[]>(`${base(id)}/bank-accounts`),
  saveBankAccount: (id: string, input: BankAccountInput, rowId?: string) =>
    rowId ? api.put<BankAccount>(`${base(id)}/bank-accounts/${rowId}`, input) : api.post<BankAccount>(`${base(id)}/bank-accounts`, input),
  deleteBankAccount: (id: string, rowId: string) => api.delete(`${base(id)}/bank-accounts/${rowId}`),

  nextOfKins: (id: string) => api.get<NextOfKin[]>(`${base(id)}/next-of-kins`),
  saveNextOfKin: (id: string, input: NextOfKinInput, rowId?: string) =>
    rowId ? api.put<NextOfKin>(`${base(id)}/next-of-kins/${rowId}`, input) : api.post<NextOfKin>(`${base(id)}/next-of-kins`, input),
  deleteNextOfKin: (id: string, rowId: string) => api.delete(`${base(id)}/next-of-kins/${rowId}`),

  documents: (id: string, kind: DocumentKind) => api.get<CrewDocument[]>(`${base(id)}/${kind}`),
  saveDocument: (id: string, kind: DocumentKind, input: CrewDocumentInput, rowId?: string) =>
    rowId ? api.put<CrewDocument>(`${base(id)}/${kind}/${rowId}`, input) : api.post<CrewDocument>(`${base(id)}/${kind}`, input),
  deleteDocument: (id: string, kind: DocumentKind, rowId: string) => api.delete(`${base(id)}/${kind}/${rowId}`),

  saveExperience: (id: string, input: ExperienceInput, rowId?: string) =>
    rowId ? api.put<Experience>(`${base(id)}/experiences/${rowId}`, input) : api.post<Experience>(`${base(id)}/experiences`, input),
  deleteExperience: (id: string, rowId: string) => api.delete(`${base(id)}/experiences/${rowId}`),

  export: (format: 'csv' | 'xlsx', filters: Query) => api.post<Job>('/crews/export', undefined, { ...filters, format }),
  importTemplate: () => api.raw('/crews/import-template'),
  importValidate: (file: File) => {
    const form = new FormData()
    form.set('file', file)
    return api.upload<ImportPreview>('/crews/import/validate', form)
  },
  importCommit: (token: string) => api.post<Job>('/crews/import/commit', { token }),
}

export function crewListUrl(query: Query): string {
  return `/crews${buildQuery(query)}`
}

export function crewToInput(c: Crew): CrewInput {
  return {
    full_name: c.full_name,
    place_of_birth: c.place_of_birth,
    date_of_birth: c.date_of_birth ?? '',
    gender: c.gender ?? '',
    nationality: c.nationality,
    religion: c.religion,
    marital_status: c.marital_status ?? '',
    height_cm: c.height_cm,
    weight_kg: c.weight_kg,
    shoe_size: c.shoe_size,
    coverall_size: c.coverall_size,
    applied_rank_id: c.applied_rank_id ?? '',
    photo_file_id: c.photo_file_id ?? '',
    cv_file_id: c.cv_file_id ?? '',
    address: c.address,
    city: c.city,
    province: c.province,
    country: c.country,
    postal_code: c.postal_code,
    home_phone: c.home_phone,
    mobile_phone: c.mobile_phone,
    email: c.email,
    nearest_airport: c.nearest_airport,
  }
}

export const crewKeys = {
  all: ['crews'] as const,
  detail: (id: string) => ['crews', 'detail', id] as const,
  tab: (id: string, tab: string) => ['crews', 'detail', id, tab] as const,
}
