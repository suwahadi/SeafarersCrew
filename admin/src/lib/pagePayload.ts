import type { PageBlock, PageDetail, PageInput, PageStatus } from '@/api/pages'
import { emptySettings, type FormBlock, type FormCard, type FormItem, type FormSettings } from './pageBlocks'

export interface PageFormValues {
  title: string
  slug: string
  status: PageStatus
  seo_title: string
  meta_description: string
  og_image_file_id: string
  in_header_menu: boolean
  in_footer_menu: boolean
  menu_order: number
  intro_html: string
  blocks: FormBlock[]
}

export const emptyPageValues: PageFormValues = {
  title: '',
  slug: '',
  status: 'draft',
  seo_title: '',
  meta_description: '',
  og_image_file_id: '',
  in_header_menu: false,
  in_footer_menu: false,
  menu_order: 0,
  intro_html: '',
  blocks: [],
}

function settingsFromApi(block: PageBlock): FormSettings {
  const base = emptySettings()
  switch (block.type) {
    case 'rich_text':
      return { ...base, html: block.settings.html }
    case 'image_text':
      return { ...base, ...block.settings }
    case 'gallery':
      return { ...base, image_ids: [...block.settings.image_ids], columns: block.settings.columns, lightbox: block.settings.lightbox }
    case 'video':
      return { ...base, url: block.settings.url, title: block.settings.title }
    case 'cta':
      return {
        ...base,
        heading: block.settings.heading,
        text: block.settings.text,
        button_label: block.settings.button_label,
        button_link: block.settings.button_link,
        background_image_file_id: block.settings.background_image_file_id ?? '',
        background_color: block.settings.background_color ?? '',
      }
    case 'stats':
      return { ...base, items: block.settings.items.map((i): FormItem => ({ number: i.number, suffix: i.suffix, label: i.label, question: '', answer_html: '' })) }
    case 'faq':
      return { ...base, items: block.settings.items.map((i): FormItem => ({ number: 0, suffix: '', label: '', question: i.question, answer_html: i.answer_html })) }
    case 'card_grid':
      return {
        ...base,
        columns: block.settings.columns,
        cards: block.settings.cards.map(
          (c): FormCard => ({ icon: c.icon ?? '', image_file_id: c.image_file_id ?? '', title: c.title, text: c.text, link: c.link ?? '' }),
        ),
      }
  }
}

export function blockToForm(block: PageBlock): FormBlock {
  return { id: block.id, type: block.type, hidden: block.hidden, settings: settingsFromApi(block) }
}

function omitEmpty<T extends object>(value: T, keys: readonly (keyof T)[]): T {
  const dropped = new Set<PropertyKey>(keys)
  return Object.fromEntries(Object.entries(value).filter(([key, v]) => !(dropped.has(key) && v === ''))) as T
}

export function formToBlock(block: FormBlock): PageBlock {
  const s = block.settings
  const head = { id: block.id, hidden: block.hidden }
  switch (block.type) {
    case 'rich_text':
      return { ...head, type: 'rich_text', settings: { html: s.html } }
    case 'image_text':
      return {
        ...head,
        type: 'image_text',
        settings: {
          image_file_id: s.image_file_id,
          heading: s.heading.trim(),
          text_html: s.text_html,
          button_label: s.button_label.trim(),
          button_link: s.button_link.trim(),
          image_position: s.image_position,
        },
      }
    case 'gallery':
      return { ...head, type: 'gallery', settings: { image_ids: [...s.image_ids], columns: s.columns, lightbox: s.lightbox } }
    case 'video':
      return { ...head, type: 'video', settings: { url: s.url.trim(), title: s.title.trim() } }
    case 'cta':
      return {
        ...head,
        type: 'cta',
        settings: omitEmpty(
          {
            heading: s.heading.trim(),
            text: s.text.trim(),
            button_label: s.button_label.trim(),
            button_link: s.button_link.trim(),
            background_image_file_id: s.background_image_file_id,
            background_color: s.background_color.trim(),
          },
          ['background_image_file_id', 'background_color'],
        ),
      }
    case 'stats':
      return { ...head, type: 'stats', settings: { items: s.items.map((i) => ({ number: i.number, suffix: i.suffix.trim(), label: i.label.trim() })) } }
    case 'faq':
      return { ...head, type: 'faq', settings: { items: s.items.map((i) => ({ question: i.question.trim(), answer_html: i.answer_html })) } }
    case 'card_grid':
      return {
        ...head,
        type: 'card_grid',
        settings: {
          columns: s.columns,
          cards: s.cards.map((c) =>
            omitEmpty({ icon: c.icon.trim(), image_file_id: c.image_file_id, title: c.title.trim(), text: c.text.trim(), link: c.link.trim() }, ['icon', 'image_file_id', 'link']),
          ),
        },
      }
  }
}

export function pageToValues(page: PageDetail): PageFormValues {
  return {
    title: page.title,
    slug: page.slug,
    status: page.status,
    seo_title: page.seo_title,
    meta_description: page.meta_description,
    og_image_file_id: page.og_image_file_id ?? '',
    in_header_menu: page.in_header_menu,
    in_footer_menu: page.in_footer_menu,
    menu_order: page.menu_order,
    intro_html: page.intro_html,
    blocks: page.is_system ? [] : page.blocks.map(blockToForm),
  }
}

export function valuesToPayload(values: PageFormValues): PageInput {
  return {
    title: values.title.trim(),
    slug: values.slug.trim(),
    status: values.status,
    seo_title: values.seo_title.trim(),
    meta_description: values.meta_description.trim(),
    og_image_file_id: values.og_image_file_id === '' ? null : values.og_image_file_id,
    in_header_menu: values.in_header_menu,
    in_footer_menu: values.in_footer_menu,
    menu_order: values.menu_order,
    intro_html: values.intro_html,
    blocks: values.blocks.map(formToBlock),
  }
}
