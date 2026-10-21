import { describe, expect, it } from 'vitest'
import type { PageBlock, PageDetail } from '@/api/pages'
import { createBlock } from './pageBlocks'
import { blockToForm, emptyPageValues, formToBlock, pageToValues, valuesToPayload } from './pagePayload'

const apiBlocks: PageBlock[] = [
  { id: 'b1', type: 'rich_text', hidden: false, settings: { html: '<p>Hello</p>' } },
  { id: 'b2', type: 'video', hidden: true, settings: { url: 'https://youtu.be/dQw4w9WgXcQ', title: 'Tour', provider: 'youtube', video_id: 'dQw4w9WgXcQ' } },
  { id: 'b3', type: 'stats', hidden: false, settings: { items: [{ number: 12, suffix: '+', label: 'Ships' }] } },
  { id: 'b4', type: 'faq', hidden: false, settings: { items: [{ question: 'Why?', answer_html: '<p>Because</p>' }] } },
  { id: 'b5', type: 'card_grid', hidden: false, settings: { columns: 2, cards: [{ icon: 'anchor', title: 'Safety', text: 'First' }] } },
  { id: 'b6', type: 'cta', hidden: false, settings: { heading: 'Join', text: '', button_label: 'Apply', button_link: '/career', background_color: '#0A2540' } },
]

describe('block conversion', () => {
  it.each(apiBlocks.map((b) => [b.type, b] as const))('round-trips a %s block through the form shape', (_type, block) => {
    const back = formToBlock(blockToForm(block))
    expect(back.id).toBe(block.id)
    expect(back.type).toBe(block.type)
    expect(back.hidden).toBe(block.hidden)
  })

  it('drops the server-only video fields', () => {
    const video = formToBlock(blockToForm(apiBlocks[1]!))
    expect(video.settings).toEqual({ url: 'https://youtu.be/dQw4w9WgXcQ', title: 'Tour' })
  })

  it('omits empty optional card and call to action fields', () => {
    const grid = formToBlock(blockToForm(apiBlocks[4]!))
    expect(grid.settings).toEqual({ columns: 2, cards: [{ icon: 'anchor', title: 'Safety', text: 'First' }] })
    const cta = createBlock('cta')
    cta.settings.heading = ' Join '
    expect(formToBlock(cta).settings).toEqual({ heading: 'Join', text: '', button_label: '', button_link: '' })
  })

  it('sends only the fields that belong to the block type', () => {
    const faq = createBlock('faq')
    faq.settings.items = [{ number: 5, suffix: 'x', label: 'unused', question: ' Why? ', answer_html: '<p>A</p>' }]
    expect(formToBlock(faq).settings).toEqual({ items: [{ question: 'Why?', answer_html: '<p>A</p>' }] })
  })
})

describe('page payload', () => {
  const detail: PageDetail = {
    id: 'p1',
    title: 'About',
    slug: 'about',
    path: '/about',
    system_key: null,
    is_system: false,
    status: 'published',
    in_header_menu: true,
    in_footer_menu: false,
    menu_order: 2,
    published_at: null,
    created_at: '',
    updated_at: '',
    intro_html: '<p>Intro</p>',
    seo_title: '',
    meta_description: '',
    og_image_file_id: null,
    og_image: null,
    blocks: apiBlocks,
    media: {},
  }

  it('turns a stored page into form values and back into the write payload', () => {
    const payload = valuesToPayload(pageToValues(detail))
    expect(payload).toMatchObject({ title: 'About', slug: 'about', status: 'published', og_image_file_id: null, menu_order: 2, in_header_menu: true })
    expect(payload.blocks.map((b) => b.id)).toEqual(['b1', 'b2', 'b3', 'b4', 'b5', 'b6'])
    expect(Object.keys(payload).sort()).toEqual(
      ['blocks', 'in_footer_menu', 'in_header_menu', 'intro_html', 'menu_order', 'meta_description', 'og_image_file_id', 'seo_title', 'slug', 'status', 'title'].sort(),
    )
  })

  it('does not load blocks of a system page', () => {
    expect(pageToValues({ ...detail, is_system: true }).blocks).toEqual([])
  })

  it('trims text and sends a null social image when none is chosen', () => {
    const payload = valuesToPayload({ ...emptyPageValues, title: '  Hello  ', slug: ' hello ' })
    expect(payload.title).toBe('Hello')
    expect(payload.slug).toBe('hello')
    expect(payload.og_image_file_id).toBeNull()
  })
})
