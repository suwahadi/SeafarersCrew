import { StatsGrid } from '@/components/ui/stats-grid'
import { list } from '@/lib/list'
import type { StatsSettings } from '@/lib/types'

export function StatsBlock({ settings }: { settings: StatsSettings }) {
  const items = list(settings.items)
  if (items.length === 0) {
    return null
  }
  return <StatsGrid items={items} />
}
