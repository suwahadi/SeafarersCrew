export type BarState = 'normal' | 'due_soon' | 'overdue'

const DAY_MS = 86_400_000

export function dayNumber(iso: string): number {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso)
  if (!m) return Number.NaN
  return Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])) / DAY_MS
}

export function addDays(iso: string, days: number): string {
  const d = new Date((dayNumber(iso) + days) * DAY_MS)
  return d.toISOString().slice(0, 10)
}

export interface BarPosition {
  left: number
  width: number
  clippedStart: boolean
  clippedEnd: boolean
  visible: boolean
}

export function barPosition(start: string, end: string, from: string, to: string): BarPosition {
  const f = dayNumber(from)
  const span = dayNumber(to) - f + 1
  const s = dayNumber(start)
  const e = dayNumber(end) + 1
  const visibleStart = Math.max(s, f)
  const visibleEnd = Math.min(e, f + span)
  if (span <= 0 || visibleEnd <= visibleStart) return { left: 0, width: 0, clippedStart: false, clippedEnd: false, visible: false }
  return {
    left: ((visibleStart - f) / span) * 100,
    width: ((visibleEnd - visibleStart) / span) * 100,
    clippedStart: s < f,
    clippedEnd: e > f + span,
    visible: true,
  }
}

export function markerPosition(date: string, from: string, to: string): number | null {
  const f = dayNumber(from)
  const span = dayNumber(to) - f + 1
  const d = dayNumber(date)
  if (d < f || d >= f + span) return null
  return ((d - f + 0.5) / span) * 100
}

export function barState(plannedSignOff: string, today: string): BarState {
  const days = dayNumber(plannedSignOff) - dayNumber(today)
  if (days < 0) return 'overdue'
  if (days <= 30) return 'due_soon'
  return 'normal'
}

export interface MonthTick {
  label: string
  left: number
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'] as const

export function monthTicks(from: string, to: string): MonthTick[] {
  const ticks: MonthTick[] = []
  const f = dayNumber(from)
  const span = dayNumber(to) - f + 1
  const start = new Date(f * DAY_MS)
  let y = start.getUTCFullYear()
  let m = start.getUTCMonth()
  if (start.getUTCDate() !== 1) m += 1
  for (;;) {
    if (m > 11) {
      m = 0
      y += 1
    }
    const day = Date.UTC(y, m, 1) / DAY_MS
    if (day >= f + span) break
    ticks.push({ label: `${MONTHS[m] ?? ''} ${y}`, left: ((day - f) / span) * 100 })
    m += 1
  }
  return ticks
}
