import Link from 'next/link'
import { Container } from '@/components/ui/container'
import { Icon } from '@/components/ui/icons'
import { SiteImage } from '@/components/ui/site-image'
import { SmartLink } from '@/components/ui/smart-link'
import { list } from '@/lib/list'
import type { Site } from '@/lib/types'

const SOCIALS = [
  { key: 'linkedin', label: 'LinkedIn' },
  { key: 'facebook', label: 'Facebook' },
  { key: 'instagram', label: 'Instagram' },
  { key: 'youtube', label: 'YouTube' },
  { key: 'x', label: 'X' },
] as const

const COLUMN_TITLE = 'mb-4 text-sm font-semibold uppercase tracking-wider text-white/70'
const FOOTER_LINK = 'text-sm text-white/90 hover:text-white hover:underline'

function phoneHref(phone: string): string {
  return `tel:${phone.replace(/[^+\d]/g, '')}`
}

export function Footer({ site }: { site: Site }) {
  const logo = site.branding.logo_dark ?? site.branding.logo_light
  const phones = list(site.contact.phones)
  const menu = list(site.menus.footer)
  const socials = SOCIALS.filter((social) => site.social[social.key] !== '')
  const address = site.contact.office_address || site.organization.address
  const email = site.contact.email || site.organization.email

  return (
    <footer className="bg-primary text-on-primary">
      <Container className="grid gap-10 py-14 md:grid-cols-2 lg:grid-cols-4">
        <div>
          <Link href="/" className="inline-block" aria-label={`${site.name} home`}>
            {logo ? (
              <SiteImage image={logo} sizes="160px" alt={site.name} className="h-10 w-auto max-w-44 object-contain" />
            ) : (
              <span translate="no" className="text-xl font-bold">
                {site.name}
              </span>
            )}
          </Link>
          {site.branding.footer_text ? <p className="mt-4 max-w-xs text-sm leading-6 text-white/80">{site.branding.footer_text}</p> : null}
        </div>
        <nav aria-label="Footer">
          <h2 className={COLUMN_TITLE}>Quick links</h2>
          <ul className="space-y-2.5">
            {menu.map((item) => (
              <li key={`${item.label}-${item.href}`}>
                <SmartLink href={item.href} newTab={item.open_in_new_tab} className={FOOTER_LINK}>
                  {item.label}
                </SmartLink>
              </li>
            ))}
          </ul>
        </nav>
        <div>
          <h2 className={COLUMN_TITLE}>Contact</h2>
          <address className="space-y-2.5 text-sm not-italic text-white/90">
            {address ? <p className="whitespace-pre-line">{address}</p> : null}
            {phones.map((phone) => (
              <p key={phone}>
                <a href={phoneHref(phone)} className="hover:underline" translate="no">
                  {phone}
                </a>
              </p>
            ))}
            {email ? (
              <p>
                <a href={`mailto:${email}`} className="break-all hover:underline">
                  {email}
                </a>
              </p>
            ) : null}
          </address>
        </div>
        <div>
          <h2 className={COLUMN_TITLE}>Follow us</h2>
          {socials.length > 0 ? (
            <ul className="flex flex-wrap gap-3">
              {socials.map((social) => (
                <li key={social.key}>
                  <a
                    href={site.social[social.key]}
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-label={`${site.name} on ${social.label}`}
                    className="inline-flex size-10 items-center justify-center rounded-full bg-white/10 transition-colors duration-150 ease-out hover:bg-white/20"
                  >
                    <Icon name={social.key} className="size-5" />
                  </a>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      </Container>
      <div className="border-t border-white/15">
        <Container className="py-5 text-sm text-white/70">
          <p>
            &copy; {new Date().getFullYear()} <span translate="no">{site.name}</span>. All rights reserved.
          </p>
        </Container>
      </div>
    </footer>
  )
}
