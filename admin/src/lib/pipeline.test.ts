import { describe, expect, it } from 'vitest'
import type { Applicant, Stage } from '@/api/applicants'
import { emptyCounts, groupByStage, isStage, moveTargets, noteRequired, validateStageNote } from './pipeline'

function applicant(id: string, stage: Stage): Applicant {
  return {
    id,
    vacancy: { id: 'VAC000001', title: 'Master', slug: 'master', status: 'open' },
    full_name: `Name ${id}`,
    email: `${id}@example.com`,
    phone: '',
    nationality: '',
    rank_applied: null,
    years_experience: null,
    stage,
    files_count: 1,
    created_at: '2026-10-01T00:00:00Z',
  }
}

describe('groupByStage', () => {
  it('places each applicant in the column of its stage and keeps order', () => {
    const groups = groupByStage([applicant('a', 'new'), applicant('b', 'interview'), applicant('c', 'new')])
    expect(groups.new.map((a) => a.id)).toEqual(['a', 'c'])
    expect(groups.interview.map((a) => a.id)).toEqual(['b'])
    expect(groups.rejected).toEqual([])
  })
})

describe('moves', () => {
  it('offers every stage except the current one', () => {
    expect(moveTargets('screening')).toEqual(['new', 'interview', 'accepted', 'rejected'])
  })

  it('starts every stage count at zero', () => {
    expect(Object.values(emptyCounts())).toEqual([0, 0, 0, 0, 0])
  })
})

describe('stage note rules', () => {
  it('requires a note only for rejected', () => {
    expect(noteRequired('rejected')).toBe(true)
    expect(noteRequired('accepted')).toBe(false)
  })

  it('rejects a blank note for rejected and accepts it elsewhere', () => {
    expect(validateStageNote('rejected', '   ')).not.toBeNull()
    expect(validateStageNote('interview', '')).toBeNull()
  })

  it('rejects notes longer than 1000 characters', () => {
    expect(validateStageNote('screening', 'x'.repeat(1001))).not.toBeNull()
    expect(validateStageNote('screening', 'x'.repeat(1000))).toBeNull()
  })

  it('recognises valid stage names', () => {
    expect(isStage('interview')).toBe(true)
    expect(isStage('hired')).toBe(false)
  })
})
