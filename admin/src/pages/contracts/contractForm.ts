import { z } from 'zod'
import type { Contract, ContractInput } from '@/api/contracts'
import { isDecimal } from '@/lib/decimal'
import { isFieldEditable, type ContractStatus } from '@/lib/contractRules'

const amount = (label: string) =>
  z
    .string()
    .trim()
    .refine((v) => isDecimal(v) && !v.startsWith('-'), `${label} must be a positive amount.`)

const optionalInt = (min: number, max: number) =>
  z
    .string()
    .trim()
    .refine((v) => v === '' || (Number.isInteger(Number(v)) && Number(v) >= min && Number(v) <= max), `This must be between ${min} and ${max}.`)

export const contractSchema = z
  .object({
    crew: z.object({ id: z.string(), name: z.string() }).nullable(),
    vessel_id: z.string().min(1, 'Select a vessel.'),
    rank_id: z.string().min(1, 'Select a rank.'),
    principal_id: z.string().min(1, 'Select a principal.'),
    employer_company_id: z.string().min(1, 'Select the employer.'),
    sign_on_date: z.string().min(1, 'The sign-on date is required.'),
    duration_months: z
      .string()
      .trim()
      .refine((v) => Number.isInteger(Number(v)) && Number(v) >= 1 && Number(v) <= 60, 'The duration must be between 1 and 60 months.'),
    planned_sign_off_date: z.string(),
    point_of_hire: z.string().max(100),
    sign_on_port: z.string().max(100),
    sign_off_port: z.string().max(100),
    currency_code: z.string().min(3, 'Select a currency.'),
    basic_salary: amount('The basic salary'),
    working_hours_week: optionalInt(0, 168),
    leave_days_month: z
      .string()
      .trim()
      .refine((v) => v === '' || (isDecimal(v) && Number(v) >= 0 && Number(v) <= 31), 'This must be between 0 and 31.'),
    probation_months: optionalInt(0, 24),
    notes: z.string().max(2000),
    allowances: z
      .array(z.object({ allowance_type_id: z.string().min(1, 'Select a type.'), value: amount('The value') }))
      .max(30, 'A contract may have at most 30 allowances.'),
  })
  .refine((v) => v.crew !== null, { path: ['crew'], message: 'Select a crew member.' })
  .refine((v) => v.planned_sign_off_date === '' || v.planned_sign_off_date >= v.sign_on_date, {
    path: ['planned_sign_off_date'],
    message: 'The planned sign-off must be on or after the sign-on date.',
  })

export type ContractValues = z.infer<typeof contractSchema>

export const emptyContract: ContractValues = {
  crew: null,
  vessel_id: '',
  rank_id: '',
  principal_id: '',
  employer_company_id: '',
  sign_on_date: '',
  duration_months: '6',
  planned_sign_off_date: '',
  point_of_hire: '',
  sign_on_port: '',
  sign_off_port: '',
  currency_code: 'USD',
  basic_salary: '',
  working_hours_week: '',
  leave_days_month: '',
  probation_months: '',
  notes: '',
  allowances: [],
}

export function contractToValues(c: Contract): ContractValues {
  return {
    crew: { id: c.crew.id, name: c.crew.name },
    vessel_id: c.vessel.id,
    rank_id: c.rank.id,
    principal_id: c.principal.id,
    employer_company_id: c.employer.id,
    sign_on_date: c.sign_on_date,
    duration_months: String(c.duration_months),
    planned_sign_off_date: c.planned_sign_off_date,
    point_of_hire: c.point_of_hire,
    sign_on_port: c.sign_on_port,
    sign_off_port: c.sign_off_port,
    currency_code: c.currency_code,
    basic_salary: c.basic_salary,
    working_hours_week: c.working_hours_week === null ? '' : String(c.working_hours_week),
    leave_days_month: c.leave_days_month ?? '',
    probation_months: c.probation_months === null ? '' : String(c.probation_months),
    notes: c.notes,
    allowances: (c.allowances ?? []).map((a) => ({ allowance_type_id: a.allowance_type_id, value: a.value })),
  }
}

function intOrNull(v: string): number | null {
  return v.trim() === '' ? null : Number(v)
}

export function valuesToInput(v: ContractValues, status?: ContractStatus, original?: Contract): ContractInput {
  const edited: ContractInput = {
    crew_id: v.crew?.id ?? '',
    vessel_id: v.vessel_id,
    rank_id: v.rank_id,
    principal_id: v.principal_id,
    employer_company_id: v.employer_company_id,
    sign_on_date: v.sign_on_date,
    duration_months: Number(v.duration_months),
    planned_sign_off_date: v.planned_sign_off_date,
    point_of_hire: v.point_of_hire.trim(),
    sign_on_port: v.sign_on_port.trim(),
    sign_off_port: v.sign_off_port.trim(),
    currency_code: v.currency_code,
    basic_salary: v.basic_salary.trim(),
    working_hours_week: intOrNull(v.working_hours_week),
    leave_days_month: v.leave_days_month.trim() === '' ? null : v.leave_days_month.trim(),
    probation_months: intOrNull(v.probation_months),
    notes: v.notes.trim(),
    allowances: v.allowances.map((a) => ({ allowance_type_id: a.allowance_type_id, value: a.value.trim() })),
  }
  if (!original) return edited
  const base = valuesToInput(contractToValues(original))
  const out: Record<string, unknown> = { ...edited }
  for (const key of Object.keys(edited) as (keyof ContractInput)[]) {
    if (!isFieldEditable(status, key)) out[key] = base[key]
  }
  return out as unknown as ContractInput
}
