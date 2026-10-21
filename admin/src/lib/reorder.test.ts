import { describe, expect, it } from 'vitest'
import { moveById, moveByIdWithinSubset, moveItem, stepById } from './reorder'

const rows = ['a', 'b', 'c', 'd'].map((id) => ({ id }))
const ids = (list: { id: string }[]) => list.map((r) => r.id)

describe('reorder helpers', () => {
  it('moves an item to a new index and ignores out-of-range moves', () => {
    expect(moveItem([1, 2, 3], 0, 2)).toEqual([2, 3, 1])
    expect(moveItem([1, 2, 3], 0, 5)).toEqual([1, 2, 3])
  })

  it('moves an item onto the position of another by id', () => {
    expect(ids(moveById(rows, 'a', 'c'))).toEqual(['b', 'c', 'a', 'd'])
  })

  it('reorders only the visible rows and keeps hidden rows where they were', () => {
    const visible = new Set(['a', 'c', 'd'])
    expect(ids(moveByIdWithinSubset(rows, visible, 'd', 'a'))).toEqual(['d', 'b', 'a', 'c'])
  })

  it('steps an item past its visible neighbour', () => {
    const visible = new Set(['a', 'c', 'd'])
    expect(ids(stepById(rows, visible, 'a', 1))).toEqual(['c', 'b', 'a', 'd'])
    expect(ids(stepById(rows, visible, 'a', -1))).toEqual(['a', 'b', 'c', 'd'])
  })
})
