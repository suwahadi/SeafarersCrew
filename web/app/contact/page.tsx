import type { Metadata } from 'next'
import { ContactForm } from '@/components/forms/ContactForm'
import { Container } from '@/components/ui/container'
import { Icon } from '@/components/ui/icons'
import { MapFacade } from '@/components/ui/map-facade'
import { PageHeader } from '@/components/ui/page-header'
import { findPage, getSite } from '@/lib/api'
import { appUrl, publicApiUrl } from '@/lib/env'
import { list } from '@/lib/list'
import { safeHref } from '@/lib/links'
import { isValidCoordinates } from '@/lib/map'
import { buildMetadata, pageTitle } from '@/lib/seo'
import { firstNonEmpty } from '@/lib/text'

const FALLBACK_TITLE = 'Contact'
const DETAIL_LINK = 'font-semibold text-primary hover:text-accent-text hover:underline'

function phoneHref(phone: string): string {
  return `tel:${phone.replace(/[^+\d]/g, '')}`
}

export async function generateMetadata(): Promise<Metadata> {
  const [site, page] = await Promise.all([getSite(), findPage('contact')])
  return buildMetadata({
    site,
    baseUrl: appUrl(),
    path: '/contact',
    title: pageTitle(site, firstNonEmpty(page?.seo_title, page?.title, FALLBACK_TITLE)),
    description: page?.meta_description,
    image: page?.og_image,
  })
}

export default async function ContactPage() {
  const [site, page] = await Promise.all([getSite(), findPage('contact')])
  const title = firstNonEmpty(page?.title, FALLBACK_TITLE)
  const address = site.contact.office_address || site.organization.address
  const email = site.contact.email || site.organization.email
  const phones = list(site.contact.phones)
  const whatsappHref = safeHref(site.contact.whatsapp?.url ?? '')
  const coordinates = isValidCoordinates(site.contact.coordinates) ? site.contact.coordinates : null

  return (
    <>
      <PageHeader
        title={title}
        introHtml={page?.intro_html}
        crumbs={[
          { name: 'Home', path: '/' },
          { name: title, path: '/contact' },
        ]}
      />
      <Container className="grid gap-12 py-10 sm:py-14 lg:grid-cols-2">
        <div className="min-w-0 space-y-8">
          <section aria-labelledby="office-heading">
            <h2 id="office-heading" className="mb-4 text-2xl font-bold text-primary">
              Our office
            </h2>
            <address className="space-y-4 not-italic">
              {address ? (
                <p className="flex items-start gap-3">
                  <Icon name="map-pin" className="mt-1 size-5 shrink-0 text-accent-text" />
                  <span className="whitespace-pre-line">{address}</span>
                </p>
              ) : null}
              {phones.map((phone) => (
                <p key={phone} className="flex items-center gap-3">
                  <Icon name="phone" className="size-5 shrink-0 text-accent-text" />
                  <a href={phoneHref(phone)} translate="no" className={DETAIL_LINK}>
                    {phone}
                  </a>
                </p>
              ))}
              {email ? (
                <p className="flex items-center gap-3">
                  <Icon name="mail" className="size-5 shrink-0 text-accent-text" />
                  <a href={`mailto:${email}`} className={`${DETAIL_LINK} break-all`}>
                    {email}
                  </a>
                </p>
              ) : null}
              {whatsappHref ? (
                <p className="flex items-center gap-3">
                  <Icon name="whatsapp" className="size-5 shrink-0 text-accent-text" />
                  <a href={whatsappHref} target="_blank" rel="noopener noreferrer" className={DETAIL_LINK}>
                    Chat on WhatsApp
                  </a>
                </p>
              ) : null}
            </address>
          </section>
          {coordinates ? <MapFacade coordinates={coordinates} address={address} /> : null}
        </div>
        <section aria-labelledby="message-heading" className="min-w-0">
          <h2 id="message-heading" className="mb-2 text-2xl font-bold text-primary">
            Send us a message
          </h2>
          <p className="mb-6 text-muted">Fields marked with an asterisk are required.</p>
          <ContactForm turnstileSiteKey={site.turnstile_site_key} apiUrl={publicApiUrl()} />
        </section>
      </Container>
    </>
  )
}
