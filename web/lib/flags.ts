import type { Language } from './types'

export const BUNDLED_FLAGS: readonly string[] = ['cn', 'de', 'es', 'fr', 'gb', 'id', 'in', 'it', 'jp', 'nl', 'ph', 'pl', 'ru', 'sg', 'th', 'tr', 'ua', 'vn']

export function flagSrc(language: Pick<Language, 'flag' | 'flag_code'>): string | null {
  if (language.flag) return language.flag.url
  const code = language.flag_code.trim().toLowerCase()
  return BUNDLED_FLAGS.includes(code) ? `/flags/${code}.svg` : null
}
