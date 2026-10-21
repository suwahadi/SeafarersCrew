import { Badge, type Tone } from '@/components/ui/Layout'
import type { Vacancy, VacancyStatus } from '@/api/vacancies'

const tones: Record<VacancyStatus, Tone> = { draft: 'neutral', open: 'green', closed: 'red' }
const labels: Record<VacancyStatus, string> = { draft: 'Draft', open: 'Open', closed: 'Closed' }

export function VacancyStatusBadge({ vacancy }: { vacancy: Pick<Vacancy, 'status' | 'is_expired'> }) {
  return (
    <span className="inline-flex items-center gap-1">
      <Badge tone={tones[vacancy.status]}>{labels[vacancy.status]}</Badge>
      {vacancy.is_expired ? <Badge tone="orange">Past closing date</Badge> : null}
    </span>
  )
}
