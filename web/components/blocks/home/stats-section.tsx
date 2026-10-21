import { Container } from '@/components/ui/container'
import { StatsGrid } from '@/components/ui/stats-grid'
import type { StatItem } from '@/lib/types'

export function StatsSection({ items }: { items: readonly StatItem[] }) {
  return (
    <section aria-label="Our figures" className="bg-primary py-14 text-on-primary">
      <Container>
        <StatsGrid items={items} inverted />
      </Container>
    </section>
  )
}
