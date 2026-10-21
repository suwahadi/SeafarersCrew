const DECIMAL_PATTERN = /^-?\d+(\.\d+)?$/

export const UNIT_SCALE = 4
const UNIT = 10 ** UNIT_SCALE

export function isDecimal(value: string): boolean {
  return DECIMAL_PATTERN.test(value.trim())
}

export function toUnits(value: string | number | null | undefined): number {
  if (value === null || value === undefined) return 0
  const text = typeof value === 'number' ? value.toFixed(UNIT_SCALE) : value.trim()
  if (!isDecimal(text)) return 0
  const negative = text.startsWith('-')
  const [intPart = '0', fracPart = ''] = text.replace('-', '').split('.')
  const units = Number(intPart) * UNIT + Number((fracPart + '0'.repeat(UNIT_SCALE)).slice(0, UNIT_SCALE))
  return negative ? -units : units
}

export function roundHalfAway(value: number): number {
  return value < 0 ? -Math.round(-value) : Math.round(value)
}

export function roundUnits(units: number, decimals: number): number {
  const step = 10 ** (UNIT_SCALE - Math.min(Math.max(decimals, 0), UNIT_SCALE))
  return roundHalfAway(units / step) * step
}

export function unitsToString(units: number, decimals: number): string {
  const rounded = roundUnits(units, decimals)
  const negative = rounded < 0
  const abs = Math.abs(rounded)
  const intPart = Math.floor(abs / UNIT).toString()
  const frac = (abs % UNIT).toString().padStart(UNIT_SCALE, '0').slice(0, decimals)
  return `${negative ? '-' : ''}${intPart}${decimals > 0 ? `.${frac}` : ''}`
}

export function mulUnits(a: number, b: number): number {
  return roundHalfAway((a * b) / UNIT)
}

export function mulInt(units: number, factor: number): number {
  return units * factor
}
