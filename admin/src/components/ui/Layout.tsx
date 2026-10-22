import type { ReactNode } from 'react'

export function PageHeader({ title, description, actions }: { title: string; description?: string; actions?: ReactNode }) {
  return (
    <div className="mb-4 flex flex-col gap-3 sm:mb-6 sm:flex-row sm:flex-wrap sm:items-start sm:justify-between sm:gap-4">
      <div className="min-w-0">
        <h1 className="text-xl font-semibold break-words text-navy-800">{title}</h1>
        {description ? <p className="mt-1 text-sm text-slate-600">{description}</p> : null}
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  )
}

export function Card({ title, actions, children, className = '' }: { title?: string; actions?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={`min-w-0 rounded-lg bg-white shadow-sm ring-1 ring-slate-200 ${className}`}>
      {title || actions ? (
        <header className="flex min-h-12 flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-4 py-2">
          {title ? <h2 className="text-sm font-semibold text-slate-800">{title}</h2> : <span />}
          {actions}
        </header>
      ) : null}
      <div className="p-3 sm:p-4">{children}</div>
    </section>
  )
}

const tones = {
  neutral: 'bg-slate-100 text-slate-700 ring-slate-200',
  green: 'bg-green-50 text-green-700 ring-green-200',
  yellow: 'bg-yellow-50 text-yellow-800 ring-yellow-200',
  orange: 'bg-orange-50 text-orange-700 ring-orange-200',
  red: 'bg-red-50 text-red-700 ring-red-200',
  darkred: 'bg-red-800 text-white ring-red-900',
  blue: 'bg-blue-50 text-blue-700 ring-blue-200',
} as const

export type Tone = keyof typeof tones

export function Badge({ tone = 'neutral', children, color }: { tone?: Tone; children: ReactNode; color?: string }) {
  if (color) {
    return (
      <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ring-slate-200" style={{ color }}>
        <span aria-hidden className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: color }} />
        {children}
      </span>
    )
  }
  return <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${tones[tone]}`}>{children}</span>
}

const expiryTones: Record<string, Tone> = { expired: 'darkred', red: 'red', orange: 'orange', yellow: 'yellow', green: 'green' }
const expiryLabels: Record<string, string> = { expired: 'Expired', red: '≤ 30 days', orange: '31–60 days', yellow: '61–90 days', green: 'Valid' }

export function ExpiryBadge({ badge, daysLeft }: { badge: string; daysLeft?: number | null }) {
  const label = daysLeft !== undefined && daysLeft !== null && badge !== 'green' && badge !== 'expired' ? `${daysLeft} days` : (expiryLabels[badge] ?? badge)
  return <Badge tone={expiryTones[badge] ?? 'neutral'}>{label}</Badge>
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="rounded-lg border border-dashed border-slate-300 p-6 text-center sm:p-8">
      <p className="text-sm font-medium text-slate-700">{title}</p>
      {children ? <div className="mt-2 text-sm text-slate-500">{children}</div> : null}
    </div>
  )
}

export function DescriptionList({ items }: { items: { label: string; value: ReactNode }[] }) {
  return (
    <dl className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
      {items.map((it) => (
        <div key={it.label} className="min-w-0">
          <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{it.label}</dt>
          <dd className="mt-0.5 text-sm break-words text-slate-900">{it.value === '' || it.value === null || it.value === undefined ? '—' : it.value}</dd>
        </div>
      ))}
    </dl>
  )
}
