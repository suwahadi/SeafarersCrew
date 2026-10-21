import { activeFilterCount, EMPTY_FILTERS, filtersFromParams, filtersToParams, filtersToQuery, seaMinDays } from './filters'

describe('crew filters', () => {
  it('round-trips through URL search params', () => {
    const f = { ...EMPTY_FILTERS, status: ['available', 'on_board'], rank_id: ['r1'], nationality: 'Indonesian', group_by: 'principal' }
    const params = filtersToParams(f)
    expect(params.toString()).toBe('status=available%2Con_board&rank_id=r1&nationality=Indonesian&group_by=principal')
    expect(filtersFromParams(params)).toEqual(f)
  })

  it('maps the panel to the API query parameters', () => {
    const q = filtersToQuery({
      ...EMPTY_FILTERS,
      status: ['available'],
      age_min: '25',
      age_max: 'abc',
      doc_validity: 'expiring',
      doc_days: '60',
      sea_rank_id: 'master',
      sea_min_years: '2',
      sea_min_months: '6',
    })
    expect(q).toMatchObject({ status: ['available'], age_min: 25, age_max: undefined, doc_validity: 'expiring', doc_days: 60, sea_rank_id: 'master', sea_min_days: 910 })
  })

  it('ignores document days without a validity filter and sea service without a rank', () => {
    const q = filtersToQuery({ ...EMPTY_FILTERS, doc_days: '30', sea_min_years: '3' })
    expect(q.doc_days).toBeUndefined()
    expect(q.sea_min_days).toBeUndefined()
    expect(q.sea_rank_id).toBe('')
  })

  it('converts years and months to days using 365-day years and 30-day months', () => {
    expect(seaMinDays('1', '0')).toBe(365)
    expect(seaMinDays('', '4')).toBe(120)
    expect(seaMinDays('', '')).toBeUndefined()
  })

  it('counts active filters without the grouping option', () => {
    expect(activeFilterCount(EMPTY_FILTERS)).toBe(0)
    expect(activeFilterCount({ ...EMPTY_FILTERS, status: ['x'], compliance: 'compliant', group_by: 'company' })).toBe(2)
  })
})
