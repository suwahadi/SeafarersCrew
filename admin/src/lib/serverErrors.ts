import { blockLabel, type FormBlock } from './pageBlocks'

export interface ErrorLine {
  path: string
  message: string
}

export interface SplitErrors {
  matched: ErrorLine[]
  unmatched: ErrorLine[]
}

const BLOCK_PATH = /^blocks\.\d+(\.|$)/

export function splitServerErrors(fields: Record<string, string>, topFields: readonly string[]): SplitErrors {
  const matched: ErrorLine[] = []
  const unmatched: ErrorLine[] = []
  for (const [path, message] of Object.entries(fields)) {
    if (topFields.includes(path) || BLOCK_PATH.test(path)) matched.push({ path, message })
    else unmatched.push({ path, message })
  }
  return { matched, unmatched }
}

export function collectErrorLines(errors: unknown): ErrorLine[] {
  const out: ErrorLine[] = []
  const walk = (node: unknown, path: string[]) => {
    if (node === null || typeof node !== 'object') return
    const record = node as Record<string, unknown>
    if (typeof record.message === 'string' && typeof record.type === 'string') {
      out.push({ path: path.filter((p) => p !== 'root').join('.'), message: record.message })
    }
    for (const [key, value] of Object.entries(record)) {
      if (key === 'ref' || key === 'message' || key === 'type' || key === 'types') continue
      walk(value, [...path, key])
    }
  }
  walk(errors, [])
  return out
}

const TOP_LABELS: Record<string, string> = {
  title: 'Title',
  slug: 'Slug',
  status: 'Status',
  seo_title: 'SEO title',
  meta_description: 'Meta description',
  og_image_file_id: 'Social image',
  in_header_menu: 'Header menu',
  in_footer_menu: 'Footer menu',
  menu_order: 'Menu order',
  intro_html: 'Intro text',
  blocks: 'Blocks',
  excerpt: 'Excerpt',
  body_html: 'Body',
  thumbnail_file_id: 'Thumbnail',
  published_at: 'Publish date',
  publish_local: 'Publish date',
}

function humanize(segment: string): string {
  const spaced = segment.replace(/_html$/, '').replace(/_file_id$/, ' image').replace(/_/g, ' ')
  return spaced.charAt(0).toUpperCase() + spaced.slice(1)
}

export function describeErrorPath(path: string, blocks: readonly Pick<FormBlock, 'type'>[]): string {
  const segments = path.split('.')
  if (segments[0] !== 'blocks' || segments.length < 2) return TOP_LABELS[path] ?? humanize(path)
  const index = Number(segments[1])
  const block = blocks[index]
  const head = block ? `Block ${index + 1} (${blockLabel(block.type)})` : `Block ${index + 1}`
  const rest = segments.slice(segments[2] === 'settings' ? 3 : 2)
  const parts: string[] = []
  for (const segment of rest) {
    if (/^\d+$/.test(segment)) {
      const last = parts.pop() ?? ''
      parts.push(`${last} ${Number(segment) + 1}`.trim())
    } else {
      parts.push(humanize(segment))
    }
  }
  return [head, ...parts].join(' › ')
}

export function summarizeErrors(lines: readonly ErrorLine[], blocks: readonly Pick<FormBlock, 'type'>[]): string[] {
  return lines.map((l) => `${describeErrorPath(l.path, blocks)}: ${l.message}`)
}

export function blockIndexesWithErrors(lines: readonly ErrorLine[]): number[] {
  const found = new Set<number>()
  for (const l of lines) {
    const m = /^blocks\.(\d+)(\.|$)/.exec(l.path)
    if (m?.[1] !== undefined) found.add(Number(m[1]))
  }
  return [...found].sort((a, b) => a - b)
}
