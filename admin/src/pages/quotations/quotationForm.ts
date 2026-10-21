import { z } from 'zod'
import type { Quotation, QuotationInput, QuotationLineInput } from '@/api/quotations'
import { isDecimal } from '@/lib/decimal'

const money = (label: string) => z.string().trim().refine((v) => isDecimal(v) && !v.startsWith('-'), `${label} must be a positive amount.`)

const intInRange = (min: number, max: number) =>
  z
    .string()
    .trim()
    .refine((v) => Number.isInteger(Number(v)) && v !== '' && Number(v) >= min && Number(v) <= max, `This must be a whole number between ${min} and ${max}.`)

const lineSchema = z
  .object({
    line_type: z.enum(['daily_rate', 'lump_sum']),
    rank_id: z.string(),
    description: z.string().max(500),
    crew_count: z.string(),
    daily_rate: z.string(),
    days: z.string(),
    amount: z.string(),
  })
  .superRefine((line, ctx) => {
    const add = (path: string, message: string) => ctx.addIssue({ code: 'custom', path: [path], message })
    if (line.line_type === 'daily_rate') {
      if (line.rank_id === '') add('rank_id', 'Select a rank.')
      if (!intInRange(1, 10000).safeParse(line.crew_count).success) add('crew_count', 'Between 1 and 10000.')
      if (!intInRange(1, 3660).safeParse(line.days).success) add('days', 'Between 1 and 3660.')
      if (!money('The rate').safeParse(line.daily_rate).success) add('daily_rate', 'Enter a positive rate.')
      return
    }
    if (line.description.trim() === '') add('description', 'Describe the lump sum.')
    if (!money('The amount').safeParse(line.amount).success) add('amount', 'Enter a positive amount.')
  })

export const quotationSchema = z
  .object({
    client_company_id: z.string().min(1, 'Select the client.'),
    vessel_id: z.string(),
    currency_code: z.string().min(3, 'Select a currency.'),
    quotation_date: z.string().min(1, 'The quotation date is required.'),
    valid_until: z.string().min(1, 'The valid-until date is required.'),
    tax_percent: z
      .string()
      .trim()
      .refine((v) => isDecimal(v) && !v.startsWith('-') && Number(v) <= 100, 'The tax must be between 0 and 100.'),
    discount_amount: money('The discount'),
    terms: z.string().max(20000),
    notes: z.string().max(2000),
    lines: z.array(lineSchema).min(1, 'Add at least one daily rate or lump sum line.').max(50, 'A quotation may have at most 50 lines.'),
  })
  .refine((v) => v.valid_until === '' || v.quotation_date === '' || v.valid_until >= v.quotation_date, {
    path: ['valid_until'],
    message: 'The valid-until date must be on or after the quotation date.',
  })

export type QuotationValues = z.infer<typeof quotationSchema>
export type QuotationLineValues = QuotationValues['lines'][number]

export const emptyDailyLine: QuotationLineValues = { line_type: 'daily_rate', rank_id: '', description: '', crew_count: '1', daily_rate: '', days: '30', amount: '' }
export const emptyLumpLine: QuotationLineValues = { line_type: 'lump_sum', rank_id: '', description: '', crew_count: '', daily_rate: '', days: '', amount: '' }

export function emptyQuotation(today: string, validUntil: string): QuotationValues {
  return {
    client_company_id: '',
    vessel_id: '',
    currency_code: 'USD',
    quotation_date: today,
    valid_until: validUntil,
    tax_percent: '0',
    discount_amount: '0',
    terms: '',
    notes: '',
    lines: [{ ...emptyDailyLine }],
  }
}

export function quotationToValues(q: Quotation): QuotationValues {
  return {
    client_company_id: q.client_company_id,
    vessel_id: q.vessel_id ?? '',
    currency_code: q.currency_code,
    quotation_date: q.quotation_date,
    valid_until: q.valid_until,
    tax_percent: q.tax_percent,
    discount_amount: q.discount_amount,
    terms: q.terms ?? '',
    notes: q.notes ?? '',
    lines: (q.lines ?? []).map((l) =>
      l.line_type === 'daily_rate'
        ? { line_type: 'daily_rate', rank_id: l.rank_id ?? '', description: l.description, crew_count: String(l.crew_count ?? ''), daily_rate: l.daily_rate ?? '', days: String(l.days ?? ''), amount: '' }
        : { ...emptyLumpLine, description: l.description, amount: l.amount },
    ),
  }
}

function lineToInput(l: QuotationLineValues): QuotationLineInput {
  if (l.line_type === 'daily_rate') {
    return { line_type: 'daily_rate', rank_id: l.rank_id, description: l.description.trim(), crew_count: Number(l.crew_count), daily_rate: l.daily_rate.trim(), days: Number(l.days) }
  }
  return { line_type: 'lump_sum', description: l.description.trim(), amount: l.amount.trim() }
}

export function valuesToInput(v: QuotationValues): QuotationInput {
  return {
    client_company_id: v.client_company_id,
    vessel_id: v.vessel_id,
    currency_code: v.currency_code,
    quotation_date: v.quotation_date,
    valid_until: v.valid_until,
    tax_percent: v.tax_percent.trim(),
    discount_amount: v.discount_amount.trim(),
    terms: v.terms,
    notes: v.notes.trim(),
    lines: v.lines.map(lineToInput),
  }
}
