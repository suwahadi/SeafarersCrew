import type { BlockType } from '@/api/pages'
import { parseVideoUrl } from './videoUrl'

export type { BlockType }

export interface FormItem {
  number: number
  suffix: string
  label: string
  question: string
  answer_html: string
}

export interface FormCard {
  icon: string
  image_file_id: string
  title: string
  text: string
  link: string
}

export interface FormSettings {
  html: string
  image_file_id: string
  heading: string
  text: string
  text_html: string
  button_label: string
  button_link: string
  image_position: 'left' | 'right'
  image_ids: string[]
  columns: number
  lightbox: boolean
  url: string
  title: string
  background_image_file_id: string
  background_color: string
  items: FormItem[]
  cards: FormCard[]
}

export interface FormBlock {
  id: string
  type: BlockType
  hidden: boolean
  settings: FormSettings
}

export interface BlockTypeInfo {
  type: BlockType
  label: string
  description: string
}

export const BLOCK_TYPES: readonly BlockTypeInfo[] = [
  { type: 'rich_text', label: 'Rich Text', description: 'Headings, lists, links and tables.' },
  { type: 'image_text', label: 'Image + Text', description: 'An image beside a heading, text and button.' },
  { type: 'gallery', label: 'Gallery', description: 'Pictures chosen from the gallery.' },
  { type: 'video', label: 'Video Embed', description: 'A YouTube or Vimeo video.' },
  { type: 'cta', label: 'CTA Banner', description: 'A call to action with a button.' },
  { type: 'stats', label: 'Stats Counter', description: 'Numbers that count up.' },
  { type: 'faq', label: 'FAQ Accordion', description: 'Questions and answers.' },
  { type: 'card_grid', label: 'Card Grid', description: 'Cards with an icon or image.' },
]

export const MAX_BLOCKS = 100
export const MAX_GALLERY_IMAGES = 60
export const MAX_STATS = 12
export const MAX_FAQ = 50
export const MAX_CARDS = 24

export function blockLabel(type: BlockType): string {
  return BLOCK_TYPES.find((b) => b.type === type)?.label ?? type
}

export function newBlockId(): string {
  return crypto.randomUUID()
}

export function emptyItem(): FormItem {
  return { number: 0, suffix: '', label: '', question: '', answer_html: '' }
}

export function emptyCard(): FormCard {
  return { icon: '', image_file_id: '', title: '', text: '', link: '' }
}

export function emptySettings(): FormSettings {
  return {
    html: '',
    image_file_id: '',
    heading: '',
    text: '',
    text_html: '',
    button_label: '',
    button_link: '',
    image_position: 'left',
    image_ids: [],
    columns: 3,
    lightbox: true,
    url: '',
    title: '',
    background_image_file_id: '',
    background_color: '',
    items: [],
    cards: [],
  }
}

export function createBlock(type: BlockType): FormBlock {
  const settings = emptySettings()
  if (type === 'stats' || type === 'faq') settings.items = [emptyItem()]
  if (type === 'card_grid') settings.cards = [emptyCard()]
  return { id: newBlockId(), type, hidden: false, settings }
}

export function duplicateBlock(block: FormBlock): FormBlock {
  const { settings } = block
  return {
    ...block,
    id: newBlockId(),
    settings: {
      ...settings,
      image_ids: [...settings.image_ids],
      items: settings.items.map((i) => ({ ...i })),
      cards: settings.cards.map((c) => ({ ...c })),
    },
  }
}

export function moveItem<T>(list: readonly T[], from: number, to: number): T[] {
  if (from === to || from < 0 || to < 0 || from >= list.length || to >= list.length) return [...list]
  const next = [...list]
  const [item] = next.splice(from, 1)
  next.splice(to, 0, item as T)
  return next
}

const TAGS = /<[^>]*>/g
const SUMMARY_LENGTH = 90

export function plainText(html: string): string {
  return html.replace(TAGS, ' ').replace(/\s+/g, ' ').trim()
}

function clip(text: string): string {
  return text.length > SUMMARY_LENGTH ? `${text.slice(0, SUMMARY_LENGTH - 1)}…` : text
}

function count(n: number, singular: string, plural = `${singular}s`): string {
  return `${n} ${n === 1 ? singular : plural}`
}

export function blockSummary(block: FormBlock): string {
  const s = block.settings
  switch (block.type) {
    case 'rich_text':
      return clip(plainText(s.html)) || 'Empty'
    case 'image_text':
      return s.heading.trim() || 'No heading'
    case 'gallery':
      return `${count(s.image_ids.length, 'image')}, ${s.columns} columns`
    case 'video': {
      const parsed = parseVideoUrl(s.url)
      return s.title.trim() || (parsed ? `${parsed.provider === 'youtube' ? 'YouTube' : 'Vimeo'} ${parsed.id}` : 'No video yet')
    }
    case 'cta':
      return s.heading.trim() || 'No heading'
    case 'stats':
      return count(s.items.length, 'stat')
    case 'faq':
      return count(s.items.length, 'question')
    case 'card_grid':
      return `${count(s.cards.length, 'card')}, ${s.columns} columns`
  }
}

export interface BlockIssue {
  path: string[]
  message: string
}

const HEX_COLOR = /^#[0-9a-fA-F]{6}$/
const ICON = /^[a-z0-9][a-z0-9-]*$/

export function isValidLink(value: string): boolean {
  const v = value.trim()
  if (v === '') return true
  if (v.startsWith('/')) return !v.startsWith('//')
  return /^(https:\/\/|mailto:|tel:)\S+$/i.test(v)
}

const LINK_MESSAGE = 'Use a path starting with /, an https:// address, mailto: or tel:.'

function tooLong(value: string, max: number, name: string): string | null {
  return value.trim().length > max ? `${name} must be at most ${max} characters.` : null
}

function collect(issues: BlockIssue[], path: string[], message: string | null): void {
  if (message) issues.push({ path, message })
}

function checkLink(issues: BlockIssue[], base: string[], label: string, link: string, field: string, labelField: string): void {
  if (!isValidLink(link)) issues.push({ path: [...base, field], message: LINK_MESSAGE })
  if (label.trim() !== '' && link.trim() === '') issues.push({ path: [...base, field], message: 'Add the button link.' })
  if (link.trim() !== '' && label.trim() === '') issues.push({ path: [...base, labelField], message: 'Add the button label.' })
}

function checkColumns(issues: BlockIssue[], columns: number): void {
  if (!Number.isInteger(columns) || columns < 2 || columns > 4) issues.push({ path: ['columns'], message: 'Choose 2, 3 or 4 columns.' })
}

export function validateBlock(block: FormBlock): BlockIssue[] {
  const s = block.settings
  const issues: BlockIssue[] = []
  switch (block.type) {
    case 'rich_text':
      if (s.html.trim() === '') issues.push({ path: ['html'], message: 'Add some content.' })
      break
    case 'image_text':
      if (s.image_file_id === '') issues.push({ path: ['image_file_id'], message: 'Upload an image.' })
      collect(issues, ['heading'], tooLong(s.heading, 160, 'The heading'))
      collect(issues, ['button_label'], tooLong(s.button_label, 60, 'The button label'))
      checkLink(issues, [], s.button_label, s.button_link, 'button_link', 'button_label')
      break
    case 'gallery':
      if (s.image_ids.length === 0) issues.push({ path: ['image_ids'], message: 'Choose at least one image.' })
      if (s.image_ids.length > MAX_GALLERY_IMAGES) issues.push({ path: ['image_ids'], message: `Choose at most ${MAX_GALLERY_IMAGES} images.` })
      checkColumns(issues, s.columns)
      break
    case 'video':
      if (s.url.trim() === '') issues.push({ path: ['url'], message: 'Enter a YouTube or Vimeo link.' })
      else if (!parseVideoUrl(s.url)) issues.push({ path: ['url'], message: 'Only YouTube and Vimeo links are supported.' })
      collect(issues, ['title'], tooLong(s.title, 160, 'The title'))
      break
    case 'cta':
      if (s.heading.trim() === '') issues.push({ path: ['heading'], message: 'The heading is required.' })
      collect(issues, ['heading'], tooLong(s.heading, 160, 'The heading'))
      collect(issues, ['text'], tooLong(s.text, 500, 'The text'))
      collect(issues, ['button_label'], tooLong(s.button_label, 60, 'The button label'))
      checkLink(issues, [], s.button_label, s.button_link, 'button_link', 'button_label')
      if (s.background_color !== '' && !HEX_COLOR.test(s.background_color)) issues.push({ path: ['background_color'], message: 'Use a colour like #0A2540.' })
      break
    case 'stats':
      if (s.items.length === 0) issues.push({ path: ['items'], message: 'Add at least one stat.' })
      if (s.items.length > MAX_STATS) issues.push({ path: ['items'], message: `Add at most ${MAX_STATS} stats.` })
      s.items.forEach((item, i) => {
        if (!Number.isInteger(item.number) || item.number < 0) issues.push({ path: ['items', String(i), 'number'], message: 'Enter a whole number.' })
        if (item.label.trim() === '') issues.push({ path: ['items', String(i), 'label'], message: 'The label is required.' })
        collect(issues, ['items', String(i), 'label'], tooLong(item.label, 80, 'The label'))
        collect(issues, ['items', String(i), 'suffix'], tooLong(item.suffix, 10, 'The suffix'))
      })
      break
    case 'faq':
      if (s.items.length === 0) issues.push({ path: ['items'], message: 'Add at least one question.' })
      if (s.items.length > MAX_FAQ) issues.push({ path: ['items'], message: `Add at most ${MAX_FAQ} questions.` })
      s.items.forEach((item, i) => {
        if (item.question.trim() === '') issues.push({ path: ['items', String(i), 'question'], message: 'The question is required.' })
        collect(issues, ['items', String(i), 'question'], tooLong(item.question, 300, 'The question'))
        if (item.answer_html.trim() === '') issues.push({ path: ['items', String(i), 'answer_html'], message: 'The answer is required.' })
      })
      break
    case 'card_grid':
      checkColumns(issues, s.columns)
      if (s.cards.length === 0) issues.push({ path: ['cards'], message: 'Add at least one card.' })
      if (s.cards.length > MAX_CARDS) issues.push({ path: ['cards'], message: `Add at most ${MAX_CARDS} cards.` })
      s.cards.forEach((card, i) => {
        const at = (field: string) => ['cards', String(i), field]
        if (card.title.trim() === '') issues.push({ path: at('title'), message: 'The title is required.' })
        collect(issues, at('title'), tooLong(card.title, 120, 'The title'))
        collect(issues, at('text'), tooLong(card.text, 500, 'The text'))
        if (card.icon !== '' && (card.icon.length > 40 || !ICON.test(card.icon))) issues.push({ path: at('icon'), message: 'Use lowercase letters, digits and hyphens, up to 40 characters.' })
        if (!isValidLink(card.link)) issues.push({ path: at('link'), message: LINK_MESSAGE })
      })
      break
  }
  return issues
}
