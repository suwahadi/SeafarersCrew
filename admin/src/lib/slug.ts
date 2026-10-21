export const SLUG_MAX_LENGTH = 80

const SPECIAL_LETTERS: Record<string, string> = {
  ß: 'ss',
  æ: 'ae',
  Æ: 'ae',
  ø: 'o',
  Ø: 'o',
  œ: 'oe',
  Œ: 'oe',
  đ: 'd',
  Đ: 'd',
  ð: 'd',
  Ð: 'd',
  þ: 'th',
  Þ: 'th',
  ł: 'l',
  Ł: 'l',
  ı: 'i',
  '&': ' and ',
}

export function makeSlug(text: string): string {
  let spelled = ''
  for (const ch of text.normalize('NFKD')) {
    if (/\p{Mn}/u.test(ch)) continue
    spelled += SPECIAL_LETTERS[ch] ?? ch
  }
  const hyphenated = spelled
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+/, '')
  return hyphenated.slice(0, SLUG_MAX_LENGTH).replace(/-+$/, '')
}

export function effectiveSlug(title: string, slug: string): string {
  return makeSlug(slug.trim() === '' ? title : slug)
}
