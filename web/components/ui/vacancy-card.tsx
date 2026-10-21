import Link from 'next/link'
import { departmentLabel, employmentLabel, formatDate } from '@/lib/format'
import type { VacancyCard as VacancyCardData } from '@/lib/types'

interface VacancyCardProps {
  vacancy: VacancyCardData
  headingLevel?: 'h2' | 'h3'
}

export function VacancyCard({ vacancy, headingLevel: Heading = 'h3' }: VacancyCardProps) {
  return (
    <article className="relative flex flex-col rounded-lg border border-line bg-white p-5 transition-shadow duration-200 ease-out hover:shadow-md">
      <div className="flex items-center justify-between gap-2">
        <p translate="no" className="text-xs font-semibold uppercase tracking-wider text-accent-text">
          {vacancy.id}
        </p>
        <p className="rounded-full bg-surface px-2.5 py-0.5 text-xs font-medium text-primary">
          {employmentLabel(vacancy.employment_type)}
        </p>
      </div>
      <Heading className="mt-2 text-lg font-bold leading-snug text-primary">
        <Link href={`/career/${vacancy.slug}`} className="after:absolute after:inset-0 hover:text-accent-text">
          {vacancy.title}
        </Link>
      </Heading>
      <dl className="mt-4 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-sm">
        <dt className="text-muted">Rank</dt>
        <dd className="font-medium text-ink">{vacancy.rank.name}</dd>
        {vacancy.vessel_type ? (
          <>
            <dt className="text-muted">Vessel</dt>
            <dd className="font-medium text-ink">{vacancy.vessel_type.name}</dd>
          </>
        ) : null}
        <dt className="text-muted">Department</dt>
        <dd className="font-medium text-ink">{departmentLabel(vacancy.department)}</dd>
        {vacancy.location ? (
          <>
            <dt className="text-muted">Location</dt>
            <dd className="font-medium text-ink">{vacancy.location}</dd>
          </>
        ) : null}
        <dt className="text-muted">Closing</dt>
        <dd className="font-medium text-ink">{vacancy.closing_date ? formatDate(vacancy.closing_date) : 'Open until filled'}</dd>
      </dl>
    </article>
  )
}
