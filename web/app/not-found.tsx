import type { Metadata } from 'next'
import { ButtonLink } from '@/components/ui/button'
import { Container } from '@/components/ui/container'

export const metadata: Metadata = {
  title: 'Page not found',
  robots: { index: false, follow: false },
}

export default function NotFound() {
  return (
    <Container className="py-20 text-center sm:py-28">
      <p className="text-sm font-semibold uppercase tracking-wider text-accent-text">Error 404</p>
      <h1 className="mt-2 text-3xl font-bold tracking-tight text-primary sm:text-4xl">Page not found</h1>
      <p className="mx-auto mt-4 max-w-lg text-muted">
        The page you are looking for does not exist or has moved. Try one of these instead.
      </p>
      <div className="mt-8 flex flex-wrap justify-center gap-3">
        <ButtonLink href="/">Home</ButtonLink>
        <ButtonLink href="/career" variant="outline">
          Career
        </ButtonLink>
        <ButtonLink href="/contact" variant="outline">
          Contact
        </ButtonLink>
      </div>
    </Container>
  )
}
