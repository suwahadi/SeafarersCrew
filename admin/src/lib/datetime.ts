const PARTS: Intl.DateTimeFormatOptions = {
  hourCycle: 'h23',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
}

const LOCAL_INPUT = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/

function zoneParts(at: Date, timeZone: string): Record<string, number> {
  const out: Record<string, number> = {}
  for (const p of new Intl.DateTimeFormat('en-US', { ...PARTS, timeZone }).formatToParts(at)) {
    if (p.type !== 'literal') out[p.type] = Number(p.value)
  }
  return out
}

function zoneOffsetMs(at: Date, timeZone: string): number {
  const p = zoneParts(at, timeZone)
  const wall = Date.UTC(p.year ?? 0, (p.month ?? 1) - 1, p.day ?? 1, p.hour ?? 0, p.minute ?? 0, p.second ?? 0)
  return wall - Math.floor(at.getTime() / 1000) * 1000
}

export function localInputToDate(value: string, timeZone: string): Date | null {
  const m = LOCAL_INPUT.exec(value)
  if (!m) return null
  const wall = Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3]), Number(m[4]), Number(m[5]))
  const first = wall - zoneOffsetMs(new Date(wall), timeZone)
  return new Date(wall - zoneOffsetMs(new Date(first), timeZone))
}

export function localInputToUtc(value: string, timeZone: string): string | null {
  const date = localInputToDate(value, timeZone)
  return date ? date.toISOString().replace(/\.\d{3}Z$/, 'Z') : null
}

export function utcToLocalInput(value: string | null | undefined, timeZone: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const p = zoneParts(date, timeZone)
  const two = (n: number | undefined) => String(n ?? 0).padStart(2, '0')
  return `${String(p.year ?? 0).padStart(4, '0')}-${two(p.month)}-${two(p.day)}T${two(p.hour)}:${two(p.minute)}`
}
