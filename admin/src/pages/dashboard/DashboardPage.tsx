import { useQueries } from '@tanstack/react-query'
import { Link } from 'react-router'
import { dashboardApi, type DashboardSummary, type ExpiringItem, type UpcomingRelief } from '@/api/dashboard'
import { errorMessage } from '@/api/client'
import { Badge, Card, EmptyState, ExpiryBadge, PageHeader } from '@/components/ui/Layout'
import { Spinner } from '@/components/ui/Spinner'
import { formatDate, formatDateTime } from '@/lib/format'
import { AllCharts } from './DashboardCharts'

function Kpi({ label, value, to, tone }: { label: string; value: number; to?: string; tone?: 'red' | 'orange' | 'yellow' | 'darkred' }) {
  const color = tone === 'darkred' ? 'text-red-800' : tone === 'red' ? 'text-red-600' : tone === 'orange' ? 'text-orange-600' : tone === 'yellow' ? 'text-yellow-700' : 'text-navy-800'
  const body = (
    <>
      <p className="text-xs font-medium tracking-wide text-slate-500 uppercase">{label}</p>
      <p className={`mt-1 text-2xl font-semibold tabular-nums ${color}`}>{value.toLocaleString('en-US')}</p>
    </>
  )
  const cls = 'block rounded-lg bg-white p-4 shadow-sm ring-1 ring-slate-200'
  return to ? (
    <Link to={to} className={`${cls} transition-colors duration-150 hover:ring-navy-200`}>
      {body}
    </Link>
  ) : (
    <div className={cls}>{body}</div>
  )
}

export function KpiCards({ s }: { s: DashboardSummary }) {
  const [t1, t2, t3] = s.thresholds
  const expiring = (k: 'expired' | 'within_1' | 'within_2' | 'within_3') => s.documents[k] + s.certificates[k]
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-8">
        <Kpi label="Total crew" value={s.total_crew} to="/crews" />
        {s.by_status.map((b) => (
          <Kpi key={b.key} label={b.label} value={b.count} to={`/crews?status=${encodeURIComponent(b.key)}`} />
        ))}
      </div>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-8">
        <Kpi label="Active contracts" value={s.active_contracts} to="/contracts?status=active" />
        <Kpi label="Due for relief (30 d)" value={s.due_for_relief} to="/rotation" tone={s.due_for_relief > 0 ? 'orange' : undefined} />
        <Kpi label="Expired items" value={expiring('expired')} to="/crews?doc_validity=expired" tone={expiring('expired') > 0 ? 'darkred' : undefined} />
        <Kpi label={`Expiring ≤ ${t1} d`} value={expiring('within_1')} to={`/crews?doc_validity=expiring&doc_days=${t1}`} tone="red" />
        <Kpi label={`Expiring ${t1 + 1}–${t2} d`} value={expiring('within_2')} to={`/crews?doc_validity=expiring&doc_days=${t2}`} tone="orange" />
        <Kpi label={`Expiring ${t2 + 1}–${t3} d`} value={expiring('within_3')} to={`/crews?doc_validity=expiring&doc_days=${t3}`} tone="yellow" />
        <Kpi label="New applicants" value={s.new_applicants} />
        <Kpi label="Unread messages" value={s.unread_messages} />
      </div>
      {s.last_backup_status === 'failed' ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700 ring-1 ring-red-200">
          The last database backup failed{s.last_backup_finished_at ? ` (${formatDateTime(s.last_backup_finished_at)})` : ''}. Check Settings → Backup.
        </p>
      ) : null}
    </div>
  )
}

export function ExpiringTable({ items }: { items: ExpiringItem[] }) {
  if (items.length === 0) return <EmptyState title="Nothing expires within the alert window." />
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <thead className="bg-slate-50 text-left text-xs font-semibold text-slate-600 uppercase">
          <tr>
            <th scope="col" className="px-3 py-2">Crew</th>
            <th scope="col" className="px-3 py-2">Rank</th>
            <th scope="col" className="px-3 py-2">Item</th>
            <th scope="col" className="px-3 py-2">Number</th>
            <th scope="col" className="px-3 py-2">Expiry</th>
            <th scope="col" className="px-3 py-2">Status</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {items.map((it) => (
            <tr key={`${it.crew_id}-${it.kind}-${it.type_name}-${it.number}`}>
              <td className="px-3 py-2">
                <Link to={`/crews/${it.crew_id}`} className="font-medium text-navy-700 hover:underline">
                  {it.crew_name}
                </Link>
                <span className="ml-1 text-xs text-slate-500">{it.crew_id}</span>
              </td>
              <td className="px-3 py-2">{it.rank_name || '—'}</td>
              <td className="px-3 py-2">
                {it.type_name} <span className="text-xs text-slate-500">({it.kind})</span>
              </td>
              <td className="px-3 py-2">{it.number}</td>
              <td className="px-3 py-2 whitespace-nowrap">{formatDate(it.expired_date)}</td>
              <td className="px-3 py-2">
                <ExpiryBadge badge={it.badge} daysLeft={it.days_left} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

const reliefTone = { overdue: 'red', due_soon: 'orange', planned: 'blue' } as const
const reliefLabel = { overdue: 'Overdue', due_soon: 'Due soon', planned: 'Relief planned' } as const

export function ReliefsTable({ items }: { items: UpcomingRelief[] }) {
  if (items.length === 0) return <EmptyState title="No reliefs due in the next 30 days." />
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <thead className="bg-slate-50 text-left text-xs font-semibold text-slate-600 uppercase">
          <tr>
            <th scope="col" className="px-3 py-2">Due</th>
            <th scope="col" className="px-3 py-2">Crew</th>
            <th scope="col" className="px-3 py-2">Vessel</th>
            <th scope="col" className="px-3 py-2">Reliever</th>
            <th scope="col" className="px-3 py-2">State</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {items.map((r) => (
            <tr key={r.contract_id}>
              <td className="px-3 py-2 whitespace-nowrap">
                {formatDate(r.due_date)}
                <span className="ml-1 text-xs text-slate-500">({r.days_left} d)</span>
              </td>
              <td className="px-3 py-2">
                <Link to={`/crews/${r.crew_id}`} className="font-medium text-navy-700 hover:underline">
                  {r.crew_name}
                </Link>
                <span className="block text-xs text-slate-500">{r.rank_name}</span>
              </td>
              <td className="px-3 py-2">
                <Link to={`/contracts/${r.contract_id}`} className="hover:underline">
                  {r.vessel_name}
                </Link>
                <span className="block text-xs text-slate-500">{r.principal_name}</span>
              </td>
              <td className="px-3 py-2">{r.reliever_name ? <Link to={`/crews/${r.reliever_id}`} className="hover:underline">{r.reliever_name}</Link> : '—'}</td>
              <td className="px-3 py-2">
                <Badge tone={reliefTone[r.state]}>{reliefLabel[r.state]}</Badge>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function DashboardPage() {
  const [summary, charts, expiring, reliefs] = useQueries({
    queries: [
      { queryKey: ['dashboard', 'summary'], queryFn: ({ signal }) => dashboardApi.summary(signal) },
      { queryKey: ['dashboard', 'charts'], queryFn: ({ signal }) => dashboardApi.charts(signal) },
      { queryKey: ['dashboard', 'expiring'], queryFn: ({ signal }) => dashboardApi.expiring(signal) },
      { queryKey: ['dashboard', 'reliefs'], queryFn: ({ signal }) => dashboardApi.reliefs(signal) },
    ],
  })
  const failed = [summary, charts, expiring, reliefs].find((q) => q.isError)

  return (
    <>
      <PageHeader title="Dashboard" description={summary.data ? `Status as of ${formatDate(summary.data.today)} (WIB)` : undefined} />
      {failed ? (
        <p role="alert" className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">
          {errorMessage(failed.error)}
        </p>
      ) : null}
      {summary.data && charts.data && expiring.data && reliefs.data ? (
        <div className="space-y-6">
          <KpiCards s={summary.data} />
          <AllCharts charts={charts.data} />
          <div className="grid gap-4 xl:grid-cols-2">
            <Card title="Expiring documents and certificates">
              <ExpiringTable items={expiring.data} />
            </Card>
            <Card title="Upcoming reliefs">
              <ReliefsTable items={reliefs.data} />
            </Card>
          </div>
        </div>
      ) : failed ? null : (
        <div className="flex justify-center p-16 text-navy-800">
          <Spinner size="lg" label="Loading dashboard" />
        </div>
      )}
    </>
  )
}
