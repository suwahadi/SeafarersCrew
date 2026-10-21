import { describe, expect, it } from 'vitest'
import { buildBulkRequests, chunk, humanizeFileName, runWithConcurrency } from './bulkUpload'

describe('humanizeFileName', () => {
  it('turns file names into readable alt text', () => {
    expect(humanizeFileName('crew-on_deck.final.JPG')).toBe('Crew on deck final')
    expect(humanizeFileName('.jpg')).toBe('')
  })
})

describe('chunk', () => {
  it('splits a list into groups of at most the given size', () => {
    expect(chunk([1, 2, 3, 4, 5], 2)).toEqual([[1, 2], [3, 4], [5]])
  })
})

describe('runWithConcurrency', () => {
  it('never runs more workers than the limit and processes every item', async () => {
    let running = 0
    let peak = 0
    const done: number[] = []
    await runWithConcurrency([1, 2, 3, 4, 5, 6, 7], 3, async (n) => {
      running += 1
      peak = Math.max(peak, running)
      await new Promise((r) => setTimeout(r, 5))
      running -= 1
      done.push(n)
    })
    expect(peak).toBe(3)
    expect(done.sort()).toEqual([1, 2, 3, 4, 5, 6, 7])
  })
})

describe('buildBulkRequests', () => {
  it('chunks more than 50 entries into several requests with shared flags', () => {
    const entries = Array.from({ length: 120 }, (_, i) => ({ fileId: `f${i}`, altText: ` Alt ${i} `, caption: '' }))
    const requests = buildBulkRequests(entries, { showOnHome: true, isActive: false })
    expect(requests.map((r) => r.items.length)).toEqual([50, 50, 20])
    expect(requests[0]).toMatchObject({ show_on_home: true, is_active: false })
    expect(requests[0]?.items[0]).toEqual({ image_file_id: 'f0', alt_text: 'Alt 0', caption: '' })
  })
})
