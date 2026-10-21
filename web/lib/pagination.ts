export type PageItem = number | 'gap'

export function totalPages(total: number, perPage: number): number {
  return perPage > 0 ? Math.max(1, Math.ceil(total / perPage)) : 1
}

export function parsePageParam(raw: string | string[] | undefined): number {
  const value = Array.isArray(raw) ? raw[0] : raw
  const parsed = Number.parseInt(value ?? '', 10)
  return Number.isFinite(parsed) && parsed >= 1 ? parsed : 1
}

export function pageItems(current: number, pages: number, siblings = 1): PageItem[] {
  const wanted = new Set<number>([1, pages, current])
  for (let offset = 1; offset <= siblings; offset += 1) {
    wanted.add(current - offset)
    wanted.add(current + offset)
  }
  const numbers = [...wanted].filter((n) => n >= 1 && n <= pages).sort((a, b) => a - b)
  const items: PageItem[] = []
  let previous = 0
  for (const n of numbers) {
    if (n - previous > 1) {
      items.push('gap')
    }
    items.push(n)
    previous = n
  }
  return items
}

export function pageHref(basePath: string, page: number): string {
  return page <= 1 ? basePath : `${basePath}?page=${String(page)}`
}
