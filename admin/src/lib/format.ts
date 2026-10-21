const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'] as const

export const DEFAULT_TIMEZONE = 'Asia/Jakarta'

export function formatDate(value: string | null | undefined): string {
  if (!value) return '—'
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(value)
  if (!m) return value
  const [, y, mo, d] = m
  return `${d}-${MONTHS[Number(mo) - 1] ?? mo}-${y}`
}

export function formatDateTime(value: string | null | undefined, timeZone = DEFAULT_TIMEZONE): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).formatToParts(date)
  const get = (type: Intl.DateTimeFormatPartTypes) => parts.find((p) => p.type === type)?.value ?? ''
  return `${get('day')}-${MONTHS[Number(get('month')) - 1] ?? ''}-${get('year')} ${get('hour')}:${get('minute')}`
}

export function todayISO(timeZone = DEFAULT_TIMEZONE): string {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone, year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date())
  return parts
}

export function formatMoney(amount: string | number | null | undefined, currency: string, decimals = 2): string {
  if (amount === null || amount === undefined || amount === '') return '—'
  const text = typeof amount === 'number' ? amount.toFixed(decimals) : amount
  const negative = text.startsWith('-')
  const [intPart = '0', fracPart = ''] = text.replace('-', '').split('.')
  const grouped = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  const frac = decimals > 0 ? `.${(fracPart + '0'.repeat(decimals)).slice(0, decimals)}` : ''
  return `${currency} ${negative ? '-' : ''}${grouped}${frac}`
}

export interface Span {
  total_days: number
  years: number
  months: number
  days: number
}

export function formatSpan(span: Span | null | undefined): string {
  if (!span) return '—'
  return `${span.years} y ${span.months} m ${span.days} d`
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

export function titleCase(value: string): string {
  return value.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}
