import Link from 'next/link'
import { CardGridBlock } from '@/components/blocks/card-grid-block'
import { Container } from '@/components/ui/container'
import { SectionHeading } from '@/components/ui/section-heading'
import type { ServicesGrid } from '@/lib/api'

export function ServicesSection({ services }: { services: ServicesGrid }) {
  return (
    <section aria-labelledby="home-services" className="bg-surface py-16 sm:py-20">
      <Container>
        <SectionHeading
          id="home-services"
          eyebrow="What we do"
          title={services.title}
          action={
            <Link href="/services" className="text-sm font-semibold text-accent-text hover:underline">
              All services
            </Link>
          }
        />
        <CardGridBlock settings={services.settings} titleLevel="h3" />
      </Container>
    </section>
  )
}
