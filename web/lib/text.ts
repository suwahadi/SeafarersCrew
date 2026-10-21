export function firstNonEmpty(...values: readonly (string | null | undefined)[]): string {
  for (const value of values) {
    if (value !== null && value !== undefined && value.trim() !== '') {
      return value
    }
  }
  return ''
}

export function excerptFromHtml(html: string, maxLength = 160): string {
  const text = html
    .replace(/<[^>]*>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/\s+/g, ' ')
    .trim()
  if (text.length <= maxLength) {
    return text
  }
  const cut = text.slice(0, maxLength - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > maxLength / 2 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
}
