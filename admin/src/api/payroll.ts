import { api, type Job, type Paginated, type Query } from './client'

export interface PayrollPeriod {
  id: string
  title: string
  period_start: string
  period_end: string
  transfer_date: string | null
  line_count: number
  total_usd: string
  total_idr: string
  created_at: string
  updated_at: string
}

export interface PayrollPeriodInput {
  title: string
  period_start: string
  period_end: string
  transfer_date: string | null
}

export interface PayrollLine {
  id: string
  line_no: number
  crew: { id: string; name: string }
  employee_no: string
  crew_name: string
  rank_code: string
  vessel_name: string
  bank_account_id: string | null
  bank_name: string
  account_name: string
  account_number: string
  amount_usd: string | null
  amount_idr: string
  transfer_date: string
  remark: string
}

export interface PayrollLineInput {
  crew_id: string
  bank_account_id: string
  amount_usd: string | null
  amount_idr: string
  transfer_date: string | null
  remark: string
}

export interface PayrollTotals {
  count: number
  usd: string
  idr: string
}

export interface PayrollFacets {
  banks: string[]
  vessels: string[]
  ranks: string[]
}

export type PayrollRowStatus = 'valid' | 'new_account' | 'error'

export interface PayrollPreviewRow {
  line: number
  status: PayrollRowStatus
  employee_no: string
  crew_name: string
  bank_name: string
  account_name: string
  account_number: string
  amount_usd: string | null
  amount_idr: string
  transfer_date: string
  remark: string
  warnings?: string[]
  errors?: Record<string, string>
}

export interface PayrollPreview {
  token: string
  total: number
  valid: number
  new_accounts: number
  errors: number
  with_warnings: number
  will_replace: number
  rows: PayrollPreviewRow[]
}

export interface PayrollImportReport {
  lines: number
  accounts_added: number
}

export type PayrollLinesPage = Paginated<PayrollLine[]> & { totals: PayrollTotals }

function withSearch(path: string, search: URLSearchParams): string {
  const qs = search.toString()
  return qs ? `${path}?${qs}` : path
}

export const payrollKeys = {
  all: ['payroll'] as const,
  period: (id: string) => ['payroll', 'period', id] as const,
  facets: (id: string) => ['payroll', 'facets', id] as const,
}

export const payrollApi = {
  listPeriods: (query: Query, signal?: AbortSignal) => api.list<PayrollPeriod[]>('/payroll-periods', query, signal),
  getPeriod: (id: string) => api.get<PayrollPeriod>(`/payroll-periods/${id}`),
  createPeriod: (input: PayrollPeriodInput) => api.post<PayrollPeriod>('/payroll-periods', input),
  updatePeriod: (id: string, input: PayrollPeriodInput) => api.put<PayrollPeriod>(`/payroll-periods/${id}`, input),
  deletePeriod: (id: string) => api.delete(`/payroll-periods/${id}`),
  lines: (id: string, search: URLSearchParams, signal?: AbortSignal) =>
    api.list<PayrollLine[]>(withSearch(`/payroll-periods/${id}/lines`, search), undefined, signal) as Promise<PayrollLinesPage>,
  facets: (id: string) => api.get<PayrollFacets>(`/payroll-periods/${id}/facets`),
  createLine: (id: string, input: PayrollLineInput) => api.post<PayrollLine>(`/payroll-periods/${id}/lines`, input),
  updateLine: (lineId: string, input: PayrollLineInput) => api.put<PayrollLine>(`/payroll-lines/${lineId}`, input),
  deleteLine: (lineId: string) => api.delete(`/payroll-lines/${lineId}`),
  validateImport: (id: string, file: File) => {
    const form = new FormData()
    form.set('file', file)
    return api.upload<PayrollPreview>(`/payroll-periods/${id}/import/validate`, form)
  },
  commitImport: (id: string, token: string) => api.post<Job>(`/payroll-periods/${id}/import/commit`, { token }),
  export: (id: string, search: URLSearchParams) => api.post<Job>(withSearch(`/payroll-periods/${id}/export`, search)),
}
