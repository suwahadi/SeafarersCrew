import { list } from '@/lib/list'
import type { HomeSectionKey, Home } from '@/lib/types'

export const HOME_SECTION_KEYS: readonly HomeSectionKey[] = [
  'hero',
  'intro',
  'stats',
  'services',
  'news',
  'vacancies',
  'gallery',
  'testimonials',
  'clients',
  'cta',
]

function isSectionKey(value: string): value is HomeSectionKey {
  return (HOME_SECTION_KEYS as readonly string[]).includes(value)
}

export function sectionHasContent(key: HomeSectionKey, home: Home, hasServices: boolean): boolean {
  switch (key) {
    case 'hero':
      return list(home.hero.slides).length > 0
    case 'intro':
      return (home.page?.intro_html ?? '').trim() !== ''
    case 'stats':
      return list(home.stats).length > 0
    case 'services':
      return hasServices
    case 'news':
      return list(home.news).length > 0
    case 'vacancies':
      return list(home.vacancies).length > 0
    case 'gallery':
      return list(home.gallery).some((item) => item.image !== null)
    case 'testimonials':
      return list(home.testimonials).length > 0
    case 'clients':
      return list(home.client_logos).some((logo) => logo.logo !== null)
    case 'cta':
      return true
  }
}

export function visibleSections(home: Home, hasServices: boolean): HomeSectionKey[] {
  const seen = new Set<HomeSectionKey>()
  for (const key of list(home.sections)) {
    if (isSectionKey(key) && sectionHasContent(key, home, hasServices)) {
      seen.add(key)
    }
  }
  return [...seen]
}
