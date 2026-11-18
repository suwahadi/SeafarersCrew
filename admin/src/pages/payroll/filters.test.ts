import { activeFilterCount, EMPTY_FILTERS, filtersFromParams, filtersToParams, toSearchParams } from './filters'

describe('payroll filters', () => {
  it('round-trips filters through URL params and keeps commas inside values', () => {
    const f = { ...EMPTY_FILTERS, bank: ['BANK A, TBK & CO', 'MANDIRI'], vessel: ['MV. NUSA 52'], has_remark: true }
    const params = filtersToParams(f)
    expect(params.getAll('bank')).toEqual(['BANK A, TBK & CO', 'MANDIRI'])
    expect(params.get('has_remark')).toBe('1')
    expect(params.has('no_usd')).toBe(false)
    expect(filtersFromParams(new URLSearchParams(params.toString()))).toEqual(f)
  })

  it('serializes array query values as repeated keys', () => {
    const search = toSearchParams({ bank: ['BANK A, TBK & CO', 'MANDIRI'], q: 'santoso', page: 2, sort: undefined, order: '' })
    expect(search.getAll('bank')).toEqual(['BANK A, TBK & CO', 'MANDIRI'])
    expect(search.get('q')).toBe('santoso')
    expect(search.get('page')).toBe('2')
    expect(search.has('sort')).toBe(false)
    expect(search.has('order')).toBe(false)
  })

  it('counts each active filter once', () => {
    expect(activeFilterCount(EMPTY_FILTERS)).toBe(0)
    expect(activeFilterCount({ ...EMPTY_FILTERS, bank: ['A', 'B'], rank: ['2E'], no_usd: true })).toBe(3)
  })
})
