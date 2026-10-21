import { api, type Query } from './client'

export type QuotationStatus = 'draft' | 'sent' | 'accepted' | 'rejected' | 'expired'

export const QUOTATION_TRANSITIONS: Record<QuotationStatus, QuotationStatus[]> = {
  draft: ['sent'],
  sent: ['accepted', 'rejected', 'expired'],
  accepted: [],
  rejected: [],
  expired: [],
}

export interface QuotationLine {
  id: string
  line_type: 'daily_rate' | 'lump_sum'
  rank_id: string | null
  rank_name: string
  description: string
  crew_count: number | null
  daily_rate: string | null
  days: number | null
  amount: string
}

export interface Quotation {
  id: string
  client_company_id: string
  client_name: string
  vessel_id: string | null
  vessel_name: string
  currency_code: string
  currency_decimals: number
  quotation_date: string
  valid_until: string
  tax_percent: string
  discount_amount: string
  subtotal: string
  tax_amount: string
  grand_total: string
  terms?: string
  notes?: string
  status: QuotationStatus
  lines?: QuotationLine[]
  created_at: string
  updated_at: string
}

export interface QuotationLineInput {
  line_type: 'daily_rate' | 'lump_sum'
  rank_id?: string
  description: string
  crew_count?: number
  daily_rate?: string
  days?: number
  amount?: string
}

export interface QuotationInput {
  client_company_id: string
  vessel_id: string
  currency_code: string
  quotation_date: string
  valid_until: string
  tax_percent: string
  discount_amount: string
  terms: string
  notes: string
  lines: QuotationLineInput[]
}

export const quotationsApi = {
  list: (query: Query, signal?: AbortSignal) => api.list<Quotation[]>('/quotations', query, signal),
  get: (id: string) => api.get<Quotation>(`/quotations/${id}`),
  create: (input: QuotationInput) => api.post<Quotation>('/quotations', input),
  update: (id: string, input: QuotationInput) => api.put<Quotation>(`/quotations/${id}`, input),
  setStatus: (id: string, status: QuotationStatus) => api.patch<Quotation>(`/quotations/${id}/status`, { status }),
  remove: (id: string) => api.delete(`/quotations/${id}`),
}
