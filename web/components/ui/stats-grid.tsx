import { CountUp } from '@/components/ui/count-up'
import type { StatItem } from '@/lib/types'

export function StatsGrid({ items, inverted = false }: { items: readonly StatItem[]; inverted?: boolean }) {
  return (
    <dl className="grid grid-cols-2 gap-8 text-center lg:grid-cols-4">
      {items.map((item, index) => (
        <div key={`${item.label}-${String(index)}`} className="flex flex-col-reverse gap-1">
          <dt className={inverted ? 'text-sm font-medium text-white/85' : 'text-sm font-medium text-muted'}>{item.label}</dt>
          <dd className={`text-4xl font-extrabold tracking-tight sm:text-5xl ${inverted ? 'text-white' : 'text-primary'}`}>
            <CountUp value={item.number} suffix={item.suffix} />
          </dd>
        </div>
      ))}
    </dl>
  )
}
