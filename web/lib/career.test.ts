import { describe, expect, it } from 'vitest'
import { careerHref, hasActiveFilters, parseCareerQuery, toVacancyQuery } from '@/lib/career'

const UUID = '01a15b9b-9bad-713d-97c1-ab02ab2dcc7b'

describe('parseCareerQuery', () => {
  it('keeps valid values and normalises whitespace', () => {
    const query = parseCareerQuery({ q: '  chief   officer ', rank: UUID.toUpperCase(), department: 'deck', location: 'Batam', page: '3' })
    expect(query).toEqual({ q: 'chief officer', rank: UUID, vessel_type: '', department: 'deck', location: 'Batam', page: 3 })
  })

  it('drops malformed uuid and unknown department instead of passing them to the API', () => {
    const query = parseCareerQuery({ rank: 'not-a-uuid', vessel_type: "1' OR 1=1", department: 'galley', page: '-4' })
    expect(query.rank).toBe('')
    expect(query.vessel_type).toBe('')
    expect(query.department).toBe('')
    expect(query.page).toBe(1)
  })

  it('uses the first value of repeated params and caps long text', () => {
    const query = parseCareerQuery({ q: ['one', 'two'], location: 'x'.repeat(500), page: ['2', '5'] })
    expect(query.q).toBe('one')
    expect(query.location).toHaveLength(100)
    expect(query.page).toBe(2)
  })
})

describe('careerHref', () => {
  it('keeps filters in the url and omits page one', () => {
    expect(careerHref(parseCareerQuery({ q: 'master', department: 'deck' }), 1)).toBe('/career?q=master&department=deck')
    expect(careerHref(parseCareerQuery({ q: 'master' }), 3)).toBe('/career?q=master&page=3')
  })

  it('returns the bare path with no filters on page one', () => {
    expect(careerHref(parseCareerQuery({}), 1)).toBe('/career')
    expect(careerHref(parseCareerQuery({}), 2)).toBe('/career?page=2')
  })
})

describe('filters helpers', () => {
  it('detects active filters and ignores page', () => {
    expect(hasActiveFilters(parseCareerQuery({ page: '4' }))).toBe(false)
    expect(hasActiveFilters(parseCareerQuery({ location: 'Batam' }))).toBe(true)
  })

  it('requests twelve vacancies per page', () => {
    expect(toVacancyQuery(parseCareerQuery({ q: 'a' })).per_page).toBe(12)
  })
})
