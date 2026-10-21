import { describe, expect, it } from 'vitest'
import { BLOCK_TYPES, blockSummary, createBlock, duplicateBlock, isValidLink, moveItem, validateBlock, type BlockType, type FormBlock } from './pageBlocks'

const ID_PATTERN = /^[A-Za-z0-9_-]{1,64}$/

describe('createBlock', () => {
  it.each(BLOCK_TYPES.map((b) => b.type))('creates a visible %s block with an allowed id', (type) => {
    const block = createBlock(type)
    expect(block.type).toBe(type)
    expect(block.hidden).toBe(false)
    expect(block.id).toMatch(ID_PATTERN)
  })

  it('gives every new block a different id', () => {
    expect(createBlock('rich_text').id).not.toBe(createBlock('rich_text').id)
  })

  it('starts list blocks with one empty entry', () => {
    expect(createBlock('stats').settings.items).toHaveLength(1)
    expect(createBlock('faq').settings.items).toHaveLength(1)
    expect(createBlock('card_grid').settings.cards).toHaveLength(1)
    expect(createBlock('rich_text').settings.items).toHaveLength(0)
  })
})

describe('duplicateBlock', () => {
  it('copies the settings under a new id without sharing nested lists', () => {
    const original = createBlock('card_grid')
    original.settings.cards[0]!.title = 'Safety'
    original.settings.image_ids = ['a']
    const copy = duplicateBlock(original)
    expect(copy.id).not.toBe(original.id)
    expect(copy.settings.cards[0]?.title).toBe('Safety')
    copy.settings.cards[0]!.title = 'Changed'
    copy.settings.image_ids.push('b')
    expect(original.settings.cards[0]?.title).toBe('Safety')
    expect(original.settings.image_ids).toEqual(['a'])
  })
})

describe('moveItem', () => {
  it('moves an element and leaves the input untouched', () => {
    const input = ['a', 'b', 'c']
    expect(moveItem(input, 0, 2)).toEqual(['b', 'c', 'a'])
    expect(moveItem(input, 2, 1)).toEqual(['a', 'c', 'b'])
    expect(input).toEqual(['a', 'b', 'c'])
  })

  it('ignores positions outside the list', () => {
    expect(moveItem(['a', 'b'], 0, 5)).toEqual(['a', 'b'])
    expect(moveItem(['a', 'b'], -1, 0)).toEqual(['a', 'b'])
  })
})

describe('blockSummary', () => {
  it('strips tags from rich text', () => {
    const block = createBlock('rich_text')
    block.settings.html = '<h2>Our <strong>fleet</strong></h2><p>Modern ships</p>'
    expect(blockSummary(block)).toBe('Our fleet Modern ships')
  })

  it('describes empty and filled blocks', () => {
    expect(blockSummary(createBlock('rich_text'))).toBe('Empty')
    const gallery = createBlock('gallery')
    gallery.settings.image_ids = ['a', 'b']
    expect(blockSummary(gallery)).toBe('2 images, 3 columns')
    const video = createBlock('video')
    video.settings.url = 'https://youtu.be/dQw4w9WgXcQ'
    expect(blockSummary(video)).toBe('YouTube dQw4w9WgXcQ')
  })
})

describe('isValidLink', () => {
  it.each(['', '/contact', 'https://example.com', 'mailto:a@b.co', 'tel:+62123'])('accepts %j', (link) => {
    expect(isValidLink(link)).toBe(true)
  })

  it.each(['//evil.com', 'http://example.com', 'javascript:alert(1)', 'contact'])('rejects %j', (link) => {
    expect(isValidLink(link)).toBe(false)
  })
})

describe('validateBlock', () => {
  function paths(type: BlockType, mutate: (b: FormBlock) => void = () => undefined): string[] {
    const block = createBlock(type)
    mutate(block)
    return validateBlock(block).map((i) => i.path.join('.'))
  }

  it('requires content for rich text and an image for image + text', () => {
    expect(paths('rich_text')).toEqual(['html'])
    expect(paths('image_text')).toEqual(['image_file_id'])
  })

  it('requires a recognised video link', () => {
    expect(paths('video')).toEqual(['url'])
    expect(paths('video', (b) => (b.settings.url = 'https://example.com/x'))).toEqual(['url'])
    expect(paths('video', (b) => (b.settings.url = 'https://youtu.be/dQw4w9WgXcQ'))).toEqual([])
  })

  it('requires at least one gallery image', () => {
    expect(paths('gallery')).toEqual(['image_ids'])
  })

  it('checks the call to action heading, link pair and color', () => {
    expect(paths('cta')).toEqual(['heading'])
    expect(
      paths('cta', (b) => {
        b.settings.heading = 'Join'
        b.settings.button_label = 'Apply'
        b.settings.background_color = 'red'
      }),
    ).toEqual(['button_link', 'background_color'])
  })

  it('points at the nested item that is wrong', () => {
    expect(paths('stats')).toEqual(['items.0.label'])
    expect(paths('faq')).toEqual(['items.0.question', 'items.0.answer_html'])
    expect(paths('card_grid', (b) => (b.settings.cards[0]!.icon = 'Bad Icon'))).toEqual(['cards.0.title', 'cards.0.icon'])
  })
})
