import type { ColumnDef } from '@tanstack/react-table'
import { useState } from 'react'
import { Link } from 'react-router'
import { rotationApi, type ReliefPlan } from '@/api/contracts'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { inputClass } from '@/components/ui/Field'
import { formatDate } from '@/lib/format'
import { StatusBadge } from '../contracts/ContractStatusBadge'

const columns: ColumnDef<ReliefPlan>[] = [
  { id: 'planned_relief_date', header: 'Relief date', accessorFn: (r) => formatDate(r.planned_relief_date), enableSorting: false },
  { id: 'vessel', header: 'Vessel', accessorFn: (r) => r.vessel.name, enableSorting: false },
  { id: 'rank', header: 'Rank', accessorFn: (r) => r.rank.name, enableSorting: false },
  { id: 'relieved', header: 'Relieved', accessorFn: (r) => r.relieved.name, enableSorting: false },
  { id: 'reliever', header: 'Reliever', accessorFn: (r) => r.reliever.name, enableSorting: false },
  { id: 'joining_port', header: 'Joining port', accessorFn: (r) => r.joining_port || '—', enableSorting: false },
  { id: 'status', header: 'Status', cell: ({ row }) => <StatusBadge status={row.original.status} />, enableSorting: false },
  {
    id: 'contract',
    header: 'Contract',
    enableSorting: false,
    cell: ({ row }) => (
      <span className="flex gap-2">
        <Link to={`/contracts/${row.original.contract_id}`} onClick={(e) => e.stopPropagation()} className="text-navy-700 hover:underline">
          {row.original.contract_id}
        </Link>
        {row.original.new_contract_id ? (
          <Link to={`/contracts/${row.original.new_contract_id}`} onClick={(e) => e.stopPropagation()} className="text-navy-700 hover:underline">
            New: {row.original.new_contract_id}
          </Link>
        ) : null}
      </span>
    ),
  },
]

export function ReliefPlanList({ onOpen }: { onOpen: (plan: ReliefPlan) => void }) {
  const [status, setStatus] = useState('')
  return (
    <div className="space-y-3">
      <select aria-label="Status" className={`h-9 w-auto ${inputClass}`} value={status} onChange={(e) => setStatus(e.target.value)}>
        <option value="">All statuses</option>
        <option value="planned">Planned</option>
        <option value="confirmed">Confirmed</option>
        <option value="completed">Completed</option>
        <option value="cancelled">Cancelled</option>
      </select>
      <DataGrid<ReliefPlan>
        id="relief-plans"
        columns={columns}
        filters={{ status }}
        searchPlaceholder="Filter this page by crew or vessel…"
        rowKey={(r) => r.id}
        onRowClick={onOpen}
        fetchPage={async (params, signal) => {
          const res = await rotationApi.plans({ status: params.status, page: params.page, per_page: params.per_page }, signal)
          const needle = (params.q ?? '').toLowerCase()
          const rows =
            needle === ''
              ? res.data
              : res.data.filter((p) => [p.vessel.name, p.relieved.name, p.reliever.name].some((v) => v.toLowerCase().includes(needle)))
          return { rows, total: res.meta.total }
        }}
      />
    </div>
  )
}
