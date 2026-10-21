import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { crewKeys, crewsApi, type ComplianceItem } from '@/api/crews'
import { TextField, SelectField } from '@/components/ui/Field'
import { Badge, Card, ExpiryBadge, type Tone } from '@/components/ui/Layout'
import { useOptions } from '@/hooks/useOptions'
import { formatDate, titleCase } from '@/lib/format'
import { ComplianceBadge } from '../badges'

const statusTone: Record<ComplianceItem['status'], Tone> = { valid: 'green', expiring: 'orange', expired: 'red', missing: 'neutral' }

export function ComplianceStatusBadge({ status }: { status: ComplianceItem['status'] }) {
  return <Badge tone={statusTone[status]}>{titleCase(status)}</Badge>
}

export function ComplianceTab({ crewId }: { crewId: string }) {
  const { options } = useOptions()
  const [rankId, setRankId] = useState('')
  const [date, setDate] = useState('')
  const q = useQuery({
    queryKey: [...crewKeys.tab(crewId, 'compliance'), rankId, date],
    queryFn: () => crewsApi.compliance(crewId, { rank_id: rankId || undefined, date: date || undefined }),
  })
  const items = q.data?.items ?? []

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end gap-4">
        <div className="w-64">
          <SelectField
            label="Check against rank"
            placeholder="Applied or current rank"
            options={options.ranks.map((r) => ({ value: r.id, label: r.name }))}
            value={rankId}
            onChange={(e) => setRankId(e.target.value)}
          />
        </div>
        <div className="w-48">
          <TextField label="Check date" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </div>
        {q.data ? (
          <div className="pb-1">
            <ComplianceBadge compliant={q.data.compliant} />
          </div>
        ) : null}
      </div>
      <Card title="Required documents and certificates">
        {q.isPending ? <p className="text-sm text-slate-500">Checking compliance…</p> : null}
        {q.data && items.length === 0 ? <p className="text-sm text-slate-500">No requirements apply for this rank.</p> : null}
        {items.length > 0 ? (
          <ul className="divide-y divide-slate-100">
            {items.map((it) => (
              <li key={`${it.kind}-${it.type_id}`} className="flex flex-wrap items-center justify-between gap-2 py-2 text-sm">
                <span>
                  <span className="font-medium text-slate-800">{it.type_name}</span>
                  <span className="ml-2 text-xs text-slate-500">{it.kind === 'document' ? 'Document' : 'Certificate'}</span>
                </span>
                <span className="flex items-center gap-2">
                  {it.expired_date ? <span className="text-slate-600">{formatDate(it.expired_date)}</span> : null}
                  {it.status !== 'missing' && it.badge ? <ExpiryBadge badge={it.badge} daysLeft={it.days_left} /> : null}
                  <ComplianceStatusBadge status={it.status} />
                </span>
              </li>
            ))}
          </ul>
        ) : null}
      </Card>
    </div>
  )
}
