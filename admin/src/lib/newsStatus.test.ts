import { describe, expect, it } from 'vitest'
import { newsState, willBeScheduled } from './newsStatus'

describe('newsState', () => {
  it.each([
    [{ status: 'draft', is_live: false }, 'draft'],
    [{ status: 'draft', is_live: true }, 'draft'],
    [{ status: 'published', is_live: true }, 'live'],
    [{ status: 'published', is_live: false }, 'scheduled'],
  ] as const)('maps %j to %s', (item, expected) => {
    expect(newsState(item)).toBe(expected)
  })
})

describe('willBeScheduled', () => {
  const now = new Date('2026-10-21T00:00:00Z')

  it('is true only for a published article dated in the future', () => {
    expect(willBeScheduled('published', new Date('2026-10-22T00:00:00Z'), now)).toBe(true)
    expect(willBeScheduled('published', new Date('2026-10-20T00:00:00Z'), now)).toBe(false)
    expect(willBeScheduled('published', null, now)).toBe(false)
    expect(willBeScheduled('draft', new Date('2026-10-22T00:00:00Z'), now)).toBe(false)
  })
})
