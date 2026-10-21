import { describe, expect, it } from 'vitest'
import { localInputToUtc, utcToLocalInput } from './datetime'

describe('datetime helpers', () => {
  it('converts a wall-clock time in a zone to UTC', () => {
    expect(localInputToUtc('2026-11-01T10:00', 'Asia/Jakarta')).toBe('2026-11-01T03:00:00Z')
    expect(localInputToUtc('2026-11-01T10:00', 'UTC')).toBe('2026-11-01T10:00:00Z')
  })

  it('respects daylight saving time in the zone', () => {
    expect(localInputToUtc('2026-07-01T12:00', 'Europe/London')).toBe('2026-07-01T11:00:00Z')
    expect(localInputToUtc('2026-01-01T12:00', 'Europe/London')).toBe('2026-01-01T12:00:00Z')
  })

  it('returns null for text that is not a date and time', () => {
    expect(localInputToUtc('', 'UTC')).toBeNull()
    expect(localInputToUtc('tomorrow', 'UTC')).toBeNull()
  })

  it('shows a UTC instant as wall-clock time in the zone', () => {
    expect(utcToLocalInput('2026-11-01T03:00:00Z', 'Asia/Jakarta')).toBe('2026-11-01T10:00')
    expect(utcToLocalInput('2026-11-01T20:30:00Z', 'Asia/Jakarta')).toBe('2026-11-02T03:30')
  })

  it('returns an empty string when there is no date', () => {
    expect(utcToLocalInput(null, 'UTC')).toBe('')
    expect(utcToLocalInput('garbage', 'UTC')).toBe('')
  })
})
