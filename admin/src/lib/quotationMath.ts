import { roundUnits, toUnits, unitsToString } from './decimal'

export interface QuotationLineInput {
  line_type: 'daily_rate' | 'lump_sum'
  crew_count?: string
  daily_rate?: string
  days?: string
  amount?: string
}

export interface QuotationTotals {
  lines: string[]
  subtotal: string
  discount: string
  tax: string
  grandTotal: string
  discountExceedsSubtotal: boolean
}

function intOrZero(value: string | undefined): number {
  const n = Number(value)
  return Number.isInteger(n) && n > 0 ? n : 0
}

export function lineUnits(line: QuotationLineInput, decimals: number): number {
  if (line.line_type === 'lump_sum') return roundUnits(toUnits(line.amount ?? ''), decimals)
  return roundUnits(toUnits(line.daily_rate ?? '') * intOrZero(line.crew_count) * intOrZero(line.days), decimals)
}

export function quotationTotals(lines: QuotationLineInput[], taxPercent: string, discount: string, decimals: number): QuotationTotals {
  const amounts = lines.map((l) => lineUnits(l, decimals))
  const subtotal = amounts.reduce((a, b) => a + b, 0)
  const disc = roundUnits(toUnits(discount), decimals)
  const taxable = subtotal - disc
  const tax = roundUnits((taxable * toUnits(taxPercent)) / 1_000_000, decimals)
  return {
    lines: amounts.map((a) => unitsToString(a, decimals)),
    subtotal: unitsToString(subtotal, decimals),
    discount: unitsToString(disc, decimals),
    tax: unitsToString(tax, decimals),
    grandTotal: unitsToString(subtotal - disc + tax, decimals),
    discountExceedsSubtotal: disc > subtotal,
  }
}
