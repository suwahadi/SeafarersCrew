export const LINK_FORMAT_MESSAGE = 'Use a path starting with /, an https:// address, mailto: or tel:.'

export function isValidLink(value: string): boolean {
  const v = value.trim()
  if (v === '' || /\s/.test(v)) return false
  if (v.startsWith('//')) return false
  if (v.startsWith('/')) return true
  if (/^https:\/\/[^/]+/i.test(v)) return true
  if (/^mailto:.+/i.test(v)) return true
  return /^tel:[+\d][\d().-]*$/i.test(v)
}

export function isHttpsUrl(value: string): boolean {
  return /^https:\/\/[^\s/]+\.[^\s/]+\S*$/i.test(value.trim())
}

export function buttonPairError(label: string, link: string): { field: 'button_label' | 'button_link'; message: string } | null {
  const hasLabel = label.trim() !== ''
  const hasLink = link.trim() !== ''
  if (hasLabel && !hasLink) return { field: 'button_link', message: 'Enter the button link or clear the label.' }
  if (hasLink && !hasLabel) return { field: 'button_label', message: 'Enter the button label or clear the link.' }
  return null
}
