export type Rgb = readonly [number, number, number]

const DEFAULT_PRIMARY = '#0B2545'
const DEFAULT_ACCENT = '#13A89E'
const WHITE = '#FFFFFF'
const SURFACE = '#F3F6F9'
const INK = '#0B1220'
const AA_TEXT_RATIO = 4.5

export function parseHex(value: string): Rgb | null {
  const match = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(value.trim())
  if (!match) {
    return null
  }
  const digits = match[1] ?? ''
  const full = digits.length === 3 ? digits.replace(/(.)/g, '$1$1') : digits
  return [
    Number.parseInt(full.slice(0, 2), 16),
    Number.parseInt(full.slice(2, 4), 16),
    Number.parseInt(full.slice(4, 6), 16),
  ]
}

export function toHex(rgb: Rgb): string {
  return `#${rgb.map((channel) => Math.round(channel).toString(16).padStart(2, '0')).join('')}`.toUpperCase()
}

export function relativeLuminance(rgb: Rgb): number {
  const [r, g, b] = rgb.map((channel) => {
    const unit = channel / 255
    return unit <= 0.03928 ? unit / 12.92 : ((unit + 0.055) / 1.055) ** 2.4
  }) as [number, number, number]
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

export function contrastRatio(a: Rgb, b: Rgb): number {
  const la = relativeLuminance(a)
  const lb = relativeLuminance(b)
  const [lighter, darker] = la >= lb ? [la, lb] : [lb, la]
  return (lighter + 0.05) / (darker + 0.05)
}

export function safeColor(value: string | undefined, fallback: string): string {
  const rgb = value ? parseHex(value) : null
  return rgb ? toHex(rgb) : fallback
}

export function readableTextOn(background: string): string {
  const bg = parseHex(background) ?? [0, 0, 0]
  const white = parseHex(WHITE) ?? [255, 255, 255]
  const ink = parseHex(INK) ?? [0, 0, 0]
  return contrastRatio(bg, white) >= contrastRatio(bg, ink) ? WHITE : INK
}

export function darkenUntilContrast(color: string, against: string, minimum = AA_TEXT_RATIO): string {
  const base = parseHex(color)
  const backdrop = parseHex(against)
  if (!base || !backdrop) {
    return color
  }
  let current: Rgb = base
  for (let step = 0; step < 40 && contrastRatio(current, backdrop) < minimum; step += 1) {
    current = [current[0] * 0.95, current[1] * 0.95, current[2] * 0.95]
  }
  return toHex(current)
}

export interface BrandTheme {
  primary: string
  accent: string
  onPrimary: string
  onAccent: string
  accentText: string
}

export function brandTheme(primary: string | undefined, accent: string | undefined): BrandTheme {
  const primaryColor = safeColor(primary, DEFAULT_PRIMARY)
  const accentColor = safeColor(accent, DEFAULT_ACCENT)
  return {
    primary: primaryColor,
    accent: accentColor,
    onPrimary: readableTextOn(primaryColor),
    onAccent: readableTextOn(accentColor),
    accentText: darkenUntilContrast(accentColor, SURFACE),
  }
}
