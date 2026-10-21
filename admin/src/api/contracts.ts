import type { ContractStatus } from '@/lib/contractRules'
import type { Span } from '@/lib/format'
import { api, type Query } from './client'

export interface Ref {
  id: string
  name: string
}

export interface ContractAllowance {
  id: string
  allowance_type_id: string
  allowance_type: string
  basis: 'fixed_month' | 'fixed_day' | 'percent_basic'
  value: string
  monthly_amount: string
}

export interface Contract {
  id: string
  crew: Ref
  vessel: Ref
  rank: Ref
  principal: Ref
  employer: Ref
  sign_on_date: string
  duration_months: number
  planned_sign_off_date: string
  actual_sign_off_date: string | null
  point_of_hire: string
  sign_on_port: string
  sign_off_port: string
  currency_code: string
  currency_decimals: number
  basic_salary: string
  working_hours_week: number | null
  leave_days_month: string | null
  probation_months: number | null
  status: ContractStatus
  termination_reason: string
  notes: string
  allowances?: ContractAllowance[]
  total_monthly_pay: string
  created_at: string
  updated_at: string
}

export interface ContractInput {
  crew_id: string
  vessel_id: string
  rank_id: string
  principal_id: string
  employer_company_id: string
  sign_on_date: string
  duration_months: number
  planned_sign_off_date: string
  point_of_hire: string
  sign_on_port: string
  sign_off_port: string
  currency_code: string
  basic_salary: string
  working_hours_week: number | null
  leave_days_month: string | null
  probation_months: number | null
  notes: string
  allowances: { allowance_type_id: string; value: string }[]
}

export type ReliefStatus = 'planned' | 'confirmed' | 'completed' | 'cancelled'

export interface ReliefPlan {
  id: string
  contract_id: string
  contract_status: ContractStatus
  vessel: Ref
  rank: Ref
  relieved: Ref
  reliever: Ref
  planned_relief_date: string
  joining_port: string
  status: ReliefStatus
  notes: string
  new_contract_id?: string
  created_at: string
  updated_at: string
}

export interface ReliefWarning {
  kind: string
  type_name: string
  expired_date: string | null
  reason: string
}

export interface ReliefResult extends ReliefPlan {
  warnings: ReliefWarning[]
}

export interface ReliefInput {
  contract_id: string
  reliever_crew_id: string
  planned_relief_date: string
  joining_port: string
  status: ReliefStatus
  notes: string
}

export interface ReliefCandidate {
  id: string
  full_name: string
  nationality: string
  age: number | null
  mobile_phone: string
  sea_service_in_rank: Span
}

export interface Suggestions {
  date: string
  candidates: ReliefCandidate[]
}

export interface TimelinePlan {
  id: string
  reliever: Ref
  planned_relief_date: string
  joining_port: string
  status: ReliefStatus
}

export interface TimelineBar {
  contract_id: string
  crew: Ref
  rank_name: string
  sign_on_date: string
  planned_sign_off_date: string
  state: 'normal' | 'due_soon' | 'overdue'
  relief_plans: TimelinePlan[]
}

export interface TimelineLane {
  vessel: Ref
  vessel_type: string
  bars: TimelineBar[]
}

export interface Timeline {
  from: string
  to: string
  today: string
  lanes: TimelineLane[]
}

export const contractsApi = {
  list: (query: Query, signal?: AbortSignal) => api.list<Contract[]>('/contracts', query, signal),
  get: (id: string) => api.get<Contract>(`/contracts/${id}`),
  create: (input: ContractInput) => api.post<Contract>('/contracts', input),
  update: (id: string, input: ContractInput) => api.put<Contract>(`/contracts/${id}`, input),
  remove: (id: string) => api.delete(`/contracts/${id}`),
  activate: (id: string) => api.post<Contract>(`/contracts/${id}/activate`),
  complete: (id: string, date: string) => api.post<Contract>(`/contracts/${id}/complete`, { actual_sign_off_date: date }),
  terminate: (id: string, date: string, reason: string) => api.post<Contract>(`/contracts/${id}/terminate`, { actual_sign_off_date: date, reason }),
}

export const rotationApi = {
  timeline: (query: Query) => api.get<Timeline>('/rotation/timeline', query),
  plans: (query: Query, signal?: AbortSignal) => api.list<ReliefPlan[]>('/relief-plans', query, signal),
  get: (id: string) => api.get<ReliefPlan>(`/relief-plans/${id}`),
  create: (input: ReliefInput) => api.post<ReliefResult>('/relief-plans', input),
  update: (id: string, input: ReliefInput) => api.put<ReliefResult>(`/relief-plans/${id}`, input),
  remove: (id: string) => api.delete(`/relief-plans/${id}`),
  complete: (id: string) => api.post<Contract>(`/relief-plans/${id}/complete`),
  suggestions: (contractId: string, date?: string) => api.get<Suggestions>('/relief-plans/suggestions', { contract_id: contractId, date }),
}
