import Link from 'next/link'
import { Container } from '@/components/ui/container'
import { SectionHeading } from '@/components/ui/section-heading'
import { VacancyCard } from '@/components/ui/vacancy-card'
import type { VacancyCard as VacancyCardData } from '@/lib/types'

export function VacanciesSection({ vacancies }: { vacancies: readonly VacancyCardData[] }) {
  return (
    <section aria-labelledby="home-vacancies" className="bg-surface py-16 sm:py-20">
      <Container>
        <SectionHeading
          id="home-vacancies"
          eyebrow="Career"
          title="Open vacancies"
          action={
            <Link href="/career" className="text-sm font-semibold text-accent-text hover:underline">
              All vacancies
            </Link>
          }
        />
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {vacancies.map((vacancy) => (
            <VacancyCard key={vacancy.id} vacancy={vacancy} />
          ))}
        </div>
      </Container>
    </section>
  )
}
