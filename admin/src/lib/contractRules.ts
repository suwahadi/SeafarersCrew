import type { AllowanceBasis } from '@/api/options'
import { mulInt, mulUnits, roundUnits, toUnits, unitsToString, UNIT_SCALE } from './decimal'

export type ContractStatus = 'draft' | 'active' | 'completed' | 'terminated'
export type { AllowanceBasis }

export const DAYS_PER_MONTH = 30

export const ACTIVE_EDITABLE_FIELDS = ['planned_sign_off_date', 'sign_on_port', 'sign_off_port', 'notes'] as const

export type ContractField =
  | 'crew_id'
  | 'vessel_id'
  | 'rank_id'
  | 'principal_id'
  | 'employer_company_id'
  | 'sign_on_date'
  | 'duration_months'
  | 'planned_sign_off_date'
  | 'point_of_hire'
  | 'sign_on_port'
  | 'sign_off_port'
  | 'currency_code'
  | 'basic_salary'
  | 'working_hours_week'
  | 'leave_days_month'
  | 'probation_months'
  | 'notes'
  | 'allowances'

export function isFieldEditable(status: ContractStatus | undefined, field: ContractField): boolean {
  if (status === undefined || status === 'draft') return true
  if (status === 'active') return (ACTIVE_EDITABLE_FIELDS as readonly string[]).includes(field)
  return false
}

export function isClosed(status: ContractStatus): boolean {
  return status === 'completed' || status === 'terminated'
}

export function addMonths(isoDate: string, months: number): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(isoDate)
  if (!m || !Number.isInteger(months)) return ''
  const year = Number(m[1])
  const month = Number(m[2]) - 1 + months
  const day = Number(m[3])
  const targetYear = year + Math.floor(month / 12)
  const targetMonth = ((month % 12) + 12) % 12
  const lastDay = new Date(Date.UTC(targetYear, targetMonth + 1, 0)).getUTCDate()
  const d = Math.min(day, lastDay)
  return `${targetYear.toString().padStart(4, '0')}-${(targetMonth + 1).toString().padStart(2, '0')}-${d.toString().padStart(2, '0')}`
}

export function monthlyAllowanceUnits(basis: AllowanceBasis, value: string, basic: string, decimals: number): number {
  const v = toUnits(value)
  let amount: number
  switch (basis) {
    case 'fixed_day':
      amount = mulInt(v, DAYS_PER_MONTH)
      break
    case 'percent_basic':
      amount = mulUnits(toUnits(basic), v) / 100
      break
    default:
      amount = v
  }
  return roundUnits(amount, Math.min(decimals, UNIT_SCALE))
}

export function monthlyAllowance(basis: AllowanceBasis, value: string, basic: string, decimals: number): string {
  return unitsToString(monthlyAllowanceUnits(basis, value, basic, decimals), decimals)
}

export function totalMonthlyPay(basic: string, allowances: { basis: AllowanceBasis; value: string }[], decimals: number): string {
  const base = roundUnits(toUnits(basic), decimals)
  const total = allowances.reduce((sum, a) => sum + monthlyAllowanceUnits(a.basis, a.value, basic, decimals), base)
  return unitsToString(total, decimals)
}
