import { addDays, barPosition, barState, markerPosition, monthTicks } from './timeline'

describe('barPosition', () => {
  const from = '2026-01-01'
  const to = '2026-01-31'

  it('positions a bar inside the range as percentages of the axis', () => {
    const p = barPosition('2026-01-11', '2026-01-20', from, to)
    expect(p.visible).toBe(true)
    expect(p.left).toBeCloseTo((10 / 31) * 100)
    expect(p.width).toBeCloseTo((10 / 31) * 100)
    expect(p.clippedStart).toBe(false)
    expect(p.clippedEnd).toBe(false)
  })

  it('clips bars that extend beyond the range', () => {
    const p = barPosition('2025-12-01', '2026-02-15', from, to)
    expect(p).toMatchObject({ left: 0, width: 100, clippedStart: true, clippedEnd: true, visible: true })
  })

  it('hides bars entirely outside the range', () => {
    expect(barPosition('2025-01-01', '2025-06-01', from, to).visible).toBe(false)
    expect(barPosition('2026-03-01', '2026-06-01', from, to).visible).toBe(false)
  })

  it('shows a one-day bar', () => {
    expect(barPosition('2026-01-05', '2026-01-05', from, to).width).toBeCloseTo(100 / 31)
  })
})

describe('markerPosition', () => {
  it('centres a marker on its day and ignores dates outside the range', () => {
    expect(markerPosition('2026-01-01', '2026-01-01', '2026-01-10')).toBeCloseTo(5)
    expect(markerPosition('2026-01-11', '2026-01-01', '2026-01-10')).toBeNull()
    expect(markerPosition('2025-12-31', '2026-01-01', '2026-01-10')).toBeNull()
  })
})

describe('barState', () => {
  it.each([
    ['2026-10-20', 'overdue'],
    ['2026-10-21', 'due_soon'],
    ['2026-11-20', 'due_soon'],
    ['2026-11-21', 'normal'],
  ])('classifies planned sign-off %s as %s relative to 2026-10-21', (planned, want) => {
    expect(barState(planned, '2026-10-21')).toBe(want)
  })
})

describe('date helpers', () => {
  it('adds days across month and year boundaries', () => {
    expect(addDays('2026-12-30', 5)).toBe('2027-01-04')
    expect(addDays('2026-03-01', -1)).toBe('2026-02-28')
  })

  it('lists month ticks inside the range', () => {
    const ticks = monthTicks('2026-01-15', '2026-04-10')
    expect(ticks.map((t) => t.label)).toEqual(['Feb 2026', 'Mar 2026', 'Apr 2026'])
    expect(ticks[0]?.left).toBeGreaterThan(0)
  })
})
