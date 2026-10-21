import { describe, expect, it } from 'vitest'
import { effectiveSlug, makeSlug } from './slug'

describe('makeSlug', () => {
  it.each([
    ['Third Officer – Bulk Carrier', 'third-officer-bulk-carrier'],
    ['  Chief   Engineer!! ', 'chief-engineer'],
    ['Café & Galley Crew', 'cafe-and-galley-crew'],
    ['Straße', 'strasse'],
    ['***', ''],
  ])('normalizes %j to %j', (input, expected) => {
    expect(makeSlug(input)).toBe(expected)
  })

  it('limits the slug to 80 characters without a trailing hyphen', () => {
    expect(makeSlug(`${'a'.repeat(79)} bbb`)).toBe('a'.repeat(79))
    expect(makeSlug('word '.repeat(40)).length).toBeLessThanOrEqual(80)
  })
})

describe('effectiveSlug', () => {
  it('derives the slug from the title when the slug field is blank', () => {
    expect(effectiveSlug('Master Mariner', '  ')).toBe('master-mariner')
  })

  it('normalizes a typed slug instead of using the title', () => {
    expect(effectiveSlug('Master Mariner', 'My Custom Slug')).toBe('my-custom-slug')
  })
})
