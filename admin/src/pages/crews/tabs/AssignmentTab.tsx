import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router'
import { crewKeys, crewsApi } from '@/api/crews'
import { DescriptionList, EmptyState } from '@/components/ui/Layout'
import { formatDate } from '@/lib/format'

export function AssignmentTab({ crewId }: { crewId: string }) {
  const q = useQuery({ queryKey: crewKeys.tab(crewId, 'assignment'), queryFn: () => crewsApi.currentAssignment(crewId) })
  if (q.isPending) return <p className="text-sm text-slate-500">Loading current assignment…</p>
  if (!q.data) return <EmptyState title="Not currently assigned">This crew member is not on board a vessel.</EmptyState>
  const a = q.data
  return (
    <div className="space-y-4">
      <DescriptionList
        items={[
          { label: 'Vessel', value: a.vessel.name },
          { label: 'Vessel type', value: a.vessel_type },
          { label: 'Rank', value: a.rank.name },
          { label: 'Principal', value: a.principal.name },
          { label: 'Sign-on date', value: formatDate(a.sign_on_date) },
          { label: 'Planned sign-off', value: formatDate(a.planned_sign_off_date) },
          { label: 'Planned relief', value: formatDate(a.planned_relief_date) },
          { label: 'Days on board', value: String(a.days_on_board) },
          { label: 'Contract', value: <Link to={`/contracts/${a.contract_id}`} className="font-mono text-navy-700 hover:underline">{a.contract_id}</Link> },
        ]}
      />
    </div>
  )
}
