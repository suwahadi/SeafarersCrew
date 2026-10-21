import { ButtonLink } from '@/components/ui/button'
import { Container } from '@/components/ui/container'

export function CtaSection({ siteName }: { siteName: string }) {
  return (
    <section aria-labelledby="home-cta" className="pb-16 pt-4 sm:pb-20">
      <Container>
        <div className="rounded-xl bg-primary px-6 py-12 text-center text-on-primary sm:px-12 sm:py-16">
          <h2 id="home-cta" className="text-2xl font-bold tracking-tight sm:text-3xl">
            Start your next contract with <span translate="no">{siteName}</span>
          </h2>
          <p className="mx-auto mt-3 max-w-2xl text-lg opacity-90">
            Browse our open vacancies and apply in minutes. Our crewing team reviews every application.
          </p>
          <ButtonLink href="/career" variant="accent" className="mt-8">
            View open vacancies
          </ButtonLink>
        </div>
      </Container>
    </section>
  )
}
