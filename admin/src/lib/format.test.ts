import { formatBytes, formatDate, formatDateTime, formatMoney, formatSpan, titleCase } from './format'

describe('formatDate', () => {
  it('renders ISO dates as DD-MMM-YYYY', () => {
    expect(formatDate('2026-10-20')).toBe('20-Oct-2026')
    expect(formatDate('2026-01-05T10:00:00Z')).toBe('05-Jan-2026')
  })

  it('renders an em dash for empty values', () => {
    expect(formatDate(null)).toBe('—')
    expect(formatDate('')).toBe('—')
  })
})

describe('formatDateTime', () => {
  it('converts UTC timestamps to WIB', () => {
    expect(formatDateTime('2026-10-20T18:30:00Z')).toBe('21-Oct-2026 01:30')
  })
})

describe('formatMoney', () => {
  it.each([
    ['45000', 'USD', 2, 'USD 45,000.00'],
    ['120000000', 'IDR', 0, 'IDR 120,000,000'],
    ['1234.5', 'EUR', 2, 'EUR 1,234.50'],
    ['-7', 'SGD', 2, 'SGD -7.00'],
    [1000, 'USD', 2, 'USD 1,000.00'],
  ])('formats %s %s with %d decimals', (amount, currency, decimals, want) => {
    expect(formatMoney(amount, currency, decimals)).toBe(want)
  })
})

describe('other formatters', () => {
  it('formats spans, bytes and labels', () => {
    expect(formatSpan({ total_days: 400, years: 1, months: 1, days: 5 })).toBe('1 y 1 m 5 d')
    expect(formatBytes(2048)).toBe('2.0 KB')
    expect(titleCase('on_board')).toBe('On Board')
  })
})
