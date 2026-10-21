import { Container } from '@/components/ui/container'
import { SectionHeading } from '@/components/ui/section-heading'
import { SiteImage } from '@/components/ui/site-image'
import { safeHref } from '@/lib/links'
import type { ClientLogo } from '@/lib/types'

function Logo({ client }: { client: ClientLogo }) {
  const image = (
    <SiteImage
      image={client.logo}
      sizes="160px"
      alt={client.name}
      className="mx-auto h-12 w-auto max-w-40 object-contain opacity-80 grayscale transition duration-200 ease-out hover:opacity-100 hover:grayscale-0"
    />
  )
  const href = safeHref(client.url)
  return href === '' ? (
    image
  ) : (
    <a href={href} target="_blank" rel="noopener noreferrer" aria-label={client.name} className="block">
      {image}
    </a>
  )
}

export function ClientsSection({ clients }: { clients: readonly ClientLogo[] }) {
  return (
    <section aria-labelledby="home-clients" className="py-16 sm:py-20">
      <Container>
        <SectionHeading id="home-clients" eyebrow="Partners" title="Trusted by shipping companies" />
        <ul className="grid grid-cols-2 items-center gap-x-8 gap-y-10 sm:grid-cols-3 lg:grid-cols-5">
          {clients
            .filter((client) => client.logo !== null)
            .map((client) => (
              <li key={client.id}>
                <Logo client={client} />
              </li>
            ))}
        </ul>
      </Container>
    </section>
  )
}
