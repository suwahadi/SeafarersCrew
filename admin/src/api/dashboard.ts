import { api } from './client'

export interface CountByKey {
  key: string
  label: string
  color?: string
  group?: string
  count: number
}

export interface ExpiryBuckets {
  expired: number
  within_1: number
  within_2: number
  within_3: number
}

export interface DashboardSummary {
  today: string
  total_crew: number
  by_status: CountByKey[]
  active_contracts: number
  documents: ExpiryBuckets
  certificates: ExpiryBuckets
  due_for_relief: number
  new_applicants: number
  unread_messages: number
  thresholds: [number, number, number]
  last_backup_status?: string
  last_backup_finished_at?: string
}

export interface MonthlyMovement {
  month: string
  sign_ons: number
  sign_offs: number
}

export interface DashboardCharts {
  by_status: CountByKey[]
  by_rank: CountByKey[]
  on_board_by_principal: CountByKey[]
  movements: MonthlyMovement[]
}

export interface ExpiringItem {
  kind: 'document' | 'certificate'
  crew_id: string
  crew_name: string
  rank_name: string
  type_name: string
  number: string
  expired_date: string
  days_left: number
  badge: string
}

export interface UpcomingRelief {
  contract_id: string
  crew_id: string
  crew_name: string
  rank_name: string
  vessel_name: string
  principal_name: string
  planned_sign_off: string
  relief_plan_id: string | null
  planned_relief_date: string | null
  relief_status: string
  reliever_id: string
  reliever_name: string
  due_date: string
  days_left: number
  state: 'overdue' | 'due_soon' | 'planned'
}

export const dashboardApi = {
  summary: (signal?: AbortSignal) => api.get<DashboardSummary>('/dashboard/summary', undefined, signal),
  charts: (signal?: AbortSignal) => api.get<DashboardCharts>('/dashboard/charts', undefined, signal),
  expiring: (signal?: AbortSignal) => api.get<ExpiringItem[]>('/dashboard/expiring', undefined, signal),
  reliefs: (signal?: AbortSignal) => api.get<UpcomingRelief[]>('/dashboard/reliefs', undefined, signal),
}
