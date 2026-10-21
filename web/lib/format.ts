const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'] as const

export function formatDate(value: string | null | undefined): string {
  if (!value) {
    return ''
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  const day = String(date.getUTCDate()).padStart(2, '0')
  const month = MONTHS[date.getUTCMonth()] ?? ''
  return `${day}-${month}-${String(date.getUTCFullYear())}`
}

export function isoDate(value: string | null | undefined): string {
  if (!value) {
    return ''
  }
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : date.toISOString()
}

export function formatNumber(value: number): string {
  return new Intl.NumberFormat('en-US').format(value)
}

const DEPARTMENTS: Record<string, string> = {
  deck: 'Deck',
  engine: 'Engine',
  catering_other: 'Catering & Other',
}

export function departmentLabel(value: string): string {
  return Object.hasOwn(DEPARTMENTS, value) ? (DEPARTMENTS[value] ?? value) : value
}

export function employmentLabel(value: string): string {
  return value === 'permanent' ? 'Permanent' : 'Contract'
}
