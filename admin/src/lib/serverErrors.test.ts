import { describe, expect, it } from 'vitest'
import { blockIndexesWithErrors, collectErrorLines, describeErrorPath, splitServerErrors, summarizeErrors } from './serverErrors'

const blocks = [{ type: 'rich_text' as const }, { type: 'stats' as const }, { type: 'faq' as const }, { type: 'card_grid' as const }]

describe('splitServerErrors', () => {
  it('keeps top-level and block keys as form paths and flags the rest', () => {
    const { matched, unmatched } = splitServerErrors(
      { title: 'bad', 'blocks.3.settings.cards.1.title': 'required', blocks: 'too many', unknown_field: 'nope' },
      ['title', 'blocks'],
    )
    expect(matched.map((l) => l.path)).toEqual(['title', 'blocks.3.settings.cards.1.title', 'blocks'])
    expect(unmatched.map((l) => l.path)).toEqual(['unknown_field'])
  })
})

describe('describeErrorPath', () => {
  it('names the block, list entry and field', () => {
    expect(describeErrorPath('blocks.3.settings.cards.1.title', blocks)).toBe('Block 4 (Card Grid) › Cards 2 › Title')
    expect(describeErrorPath('blocks.1.settings.items.0.label', blocks)).toBe('Block 2 (Stats Counter) › Items 1 › Label')
    expect(describeErrorPath('blocks.2.settings.items.0.answer_html', blocks)).toBe('Block 3 (FAQ Accordion) › Items 1 › Answer')
  })

  it('uses friendly names for page fields', () => {
    expect(describeErrorPath('meta_description', blocks)).toBe('Meta description')
    expect(describeErrorPath('og_image_file_id', blocks)).toBe('Social image')
  })
})

describe('summarizeErrors and blockIndexesWithErrors', () => {
  it('joins the description and the message and lists the blocks involved', () => {
    const lines = [
      { path: 'title', message: 'The title is required.' },
      { path: 'blocks.1.settings.items.0.label', message: 'The label is required.' },
      { path: 'blocks.1.settings.items.1.label', message: 'The label is required.' },
      { path: 'blocks.0.settings.html', message: 'Add some content.' },
    ]
    expect(summarizeErrors(lines.slice(0, 2), blocks)).toEqual(['Title: The title is required.', 'Block 2 (Stats Counter) › Items 1 › Label: The label is required.'])
    expect(blockIndexesWithErrors(lines)).toEqual([0, 1])
  })
})

describe('collectErrorLines', () => {
  it('flattens nested form errors and drops the root marker', () => {
    const element = document.createElement('input')
    const lines = collectErrorLines({
      title: { type: 'required', message: 'Required.', ref: element },
      blocks: [undefined, { settings: { items: { root: { type: 'custom', message: 'Add at least one stat.' }, 1: { label: { type: 'server', message: 'Too long.' } } } } }],
    })
    expect(lines).toEqual([
      { path: 'title', message: 'Required.' },
      { path: 'blocks.1.settings.items.1.label', message: 'Too long.' },
      { path: 'blocks.1.settings.items', message: 'Add at least one stat.' },
    ])
  })
})
