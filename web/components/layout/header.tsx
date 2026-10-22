import Link from 'next/link'
import { LanguageSwitcher } from '@/components/layout/language-switcher'
import { MobileMenu } from '@/components/layout/mobile-menu'
import { Container } from '@/components/ui/container'
import { Icon } from '@/components/ui/icons'
import { SiteImage } from '@/components/ui/site-image'
import { SmartLink } from '@/components/ui/smart-link'
import { list } from '@/lib/list'
import type { MenuItem, Site } from '@/lib/types'

const LINK =
  'inline-flex items-center gap-1 rounded-md px-3 py-2 text-sm font-medium text-ink transition-colors duration-150 ease-out hover:text-accent-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-text'

function DesktopItem({ item }: { item: MenuItem }) {
  const children = list(item.children)
  return (
    <li className="nav-item relative">
      <SmartLink href={item.href} newTab={item.open_in_new_tab} className={LINK}>
        {item.label}
        {children.length > 0 ? <Icon name="chevron-down" className="size-4" /> : null}
      </SmartLink>
      {children.length > 0 ? (
        <ul className="nav-sub absolute left-0 top-full z-50 min-w-52 rounded-md border border-line bg-white p-1 shadow-lg">
          {children.map((child) => (
            <li key={`${child.label}-${child.href}`}>
              <SmartLink
                href={child.href}
                newTab={child.open_in_new_tab}
                className="block rounded px-3 py-2 text-sm text-ink hover:bg-surface hover:text-accent-text focus-visible:bg-surface"
              >
                {child.label}
              </SmartLink>
            </li>
          ))}
        </ul>
      ) : null}
    </li>
  )
}

function MobileItem({ item }: { item: MenuItem }) {
  const children = list(item.children)
  return (
    <li>
      <SmartLink
        href={item.href}
        newTab={item.open_in_new_tab}
        className="block px-4 py-3 text-base font-medium text-ink hover:bg-surface"
      >
        {item.label}
      </SmartLink>
      {children.length > 0 ? (
        <ul className="bg-surface">
          {children.map((child) => (
            <li key={`${child.label}-${child.href}`}>
              <SmartLink
                href={child.href}
                newTab={child.open_in_new_tab}
                className="block py-2.5 pl-8 pr-4 text-sm text-muted hover:text-ink"
              >
                {child.label}
              </SmartLink>
            </li>
          ))}
        </ul>
      ) : null}
    </li>
  )
}

interface HeaderProps {
  site: Site
  activeLanguage: string
}

export function Header({ site, activeLanguage }: HeaderProps) {
  const menu = list(site.menus.header)
  const logo = site.branding.logo_light ?? site.branding.logo_dark
  const languages = list(site.languages)
  const source = languages.find((language) => language.is_source)?.code ?? site.default_language

  return (
    <header className="sticky top-0 z-40 border-b border-line bg-white">
      <Container className="relative flex h-16 items-center justify-between gap-2 sm:gap-4 lg:h-20">
        <Link href="/" className="flex shrink-0 items-center" aria-label={`${site.name} home`}>
          {logo ? (
            <SiteImage image={logo} sizes="160px" eager alt={site.name} className="h-9 w-auto max-w-28 object-contain sm:max-w-44 lg:h-11" />
          ) : (
            <span translate="no" className="text-base font-bold text-primary min-[400px]:text-xl">
              {site.name}
            </span>
          )}
        </Link>
        <nav aria-label="Main" className="hidden lg:block">
          <ul className="flex items-center gap-1">
            {menu.map((item) => (
              <DesktopItem key={`${item.label}-${item.href}`} item={item} />
            ))}
          </ul>
        </nav>
        <div className="flex items-center gap-1.5 sm:gap-2">
          <LanguageSwitcher languages={[...languages]} sourceCode={source} initialCode={activeLanguage} />
          <MobileMenu>
            <nav aria-label="Mobile">
              <ul className="divide-y divide-line">
                {menu.map((item) => (
                  <MobileItem key={`${item.label}-${item.href}`} item={item} />
                ))}
              </ul>
            </nav>
          </MobileMenu>
        </div>
      </Container>
    </header>
  )
}
