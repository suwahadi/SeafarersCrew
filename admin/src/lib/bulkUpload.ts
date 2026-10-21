export const BULK_MAX_ITEMS = 50
export const UPLOAD_CONCURRENCY = 3

export function chunk<T>(items: readonly T[], size: number): T[][] {
  const out: T[][] = []
  for (let i = 0; i < items.length; i += size) out.push(items.slice(i, i + size))
  return out
}

export async function runWithConcurrency<T>(items: readonly T[], limit: number, worker: (item: T, index: number) => Promise<void>): Promise<void> {
  let next = 0
  const lane = async (): Promise<void> => {
    while (next < items.length) {
      const index = next
      next += 1
      const item = items[index]
      if (item !== undefined) await worker(item, index)
    }
  }
  await Promise.all(Array.from({ length: Math.max(1, Math.min(limit, items.length)) }, lane))
}

export function humanizeFileName(name: string): string {
  const base = name
    .replace(/\.[^./\\]+$/, '')
    .replace(/[-_.]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  if (base === '') return ''
  return base.charAt(0).toUpperCase() + base.slice(1)
}

export interface BulkEntry {
  fileId: string
  altText: string
  caption: string
}

export interface BulkRequest {
  items: { image_file_id: string; alt_text: string; caption: string }[]
  show_on_home: boolean
  is_active: boolean
}

export function buildBulkRequests(entries: readonly BulkEntry[], options: { showOnHome: boolean; isActive: boolean }): BulkRequest[] {
  return chunk(entries, BULK_MAX_ITEMS).map((group) => ({
    items: group.map((e) => ({ image_file_id: e.fileId, alt_text: e.altText.trim(), caption: e.caption.trim() })),
    show_on_home: options.showOnHome,
    is_active: options.isActive,
  }))
}
