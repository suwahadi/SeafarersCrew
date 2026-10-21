const SAFE_SCHEMES = ['http:', 'https:', 'mailto:', 'tel:']

export function safeHref(href: string): string {
  const value = href.trim()
  if (value === '') {
    return ''
  }
  if (value.startsWith('/') && !value.startsWith('//')) {
    return value
  }
  if (value.startsWith('#')) {
    return value
  }
  try {
    const url = new URL(value)
    return SAFE_SCHEMES.includes(url.protocol) ? value : ''
  } catch {
    return ''
  }
}

export function isInternalHref(href: string): boolean {
  return href.startsWith('/') || href.startsWith('#')
}
