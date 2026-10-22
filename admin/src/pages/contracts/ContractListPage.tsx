import type { ColumnDef } from '@tanstack/react-table'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { contractsApi, type Contract } from '@/api/contracts'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { FilterField, filterClass } from '@/components/ui/Field'
import { PageHeader } from '@/components/ui/Layout'
import { useOptions } from '@/hooks/useOptions'
import { formatDate, formatMoney } from '@/lib/format'
import { StatusBadge } from './ContractStatusBadge'

const columns: ColumnDef<Contract>[] = [
  {
    id: 'id',
    accessorFn: (r) => r.id,
    header: 'Contract ID',
    cell: ({ row }) => (
      <Link to={`/contracts/${row.original.id}`} className="font-medium text-navy-700 hover:underline" onClick={(e) => e.stopPropagation()}>
        {row.original.id}
      </Link>
    ),
    enableHiding: false,
  },
  { id: 'crew_name', accessorFn: (r) => r.crew.name, header: 'Crew', cell: ({ row }) => `${row.original.crew.name} (${row.original.crew.id})` },
  { id: 'rank', accessorFn: (r) => r.rank.name, header: 'Rank', cell: ({ row }) => row.original.rank.name },
  { id: 'vessel', accessorFn: (r) => r.vessel.name, header: 'Vessel', cell: ({ row }) => row.original.vessel.name },
  { id: 'principal', accessorFn: (r) => r.principal.name, header: 'Principal', cell: ({ row }) => row.original.principal.name },
  { id: 'sign_on_date', accessorFn: (r) => r.sign_on_date, header: 'Sign-on', cell: ({ row }) => formatDate(row.original.sign_on_date) },
  { id: 'planned_sign_off_date', accessorFn: (r) => r.planned_sign_off_date, header: 'Planned sign-off', cell: ({ row }) => formatDate(row.original.planned_sign_off_date) },
  {
    id: 'basic_salary',
    accessorFn: (r) => r.basic_salary,
    header: 'Basic salary',
    cell: ({ row }) => formatMoney(row.original.basic_salary, row.original.currency_code, row.original.currency_decimals),
  },
  { id: 'status', accessorFn: (r) => r.status, header: 'Status', cell: ({ row }) => <StatusBadge status={row.original.status} /> },
  { id: 'created_at', accessorFn: (r) => r.created_at, header: 'Created', cell: ({ row }) => formatDate(row.original.created_at) },
]

const statuses = [
  { value: '', label: 'All statuses' },
  { value: 'draft', label: 'Draft' },
  { value: 'active', label: 'Active' },
  { value: 'completed', label: 'Completed' },
  { value: 'terminated', label: 'Terminated' },
]

export default function ContractListPage() {
  const navigate = useNavigate()
  const { options } = useOptions()
  const [filters, setFilters] = useState<Record<string, string>>({})
  const set = (key: string, value: string) => setFilters((f) => ({ ...f, [key]: value }))
  const principals = options.companies.filter((c) => c.type !== 'company')
  const select = filterClass

  return (
    <>
      <PageHeader
        title="Contracts"
        description="Seafarer employment contracts with salary, allowances and lifecycle."
        actions={
          <Button icon={<Icon name="plus" />} onClick={() => void navigate('/contracts/new')}>
            New contract
          </Button>
        }
      />
      <div className="filter-bar mb-3" role="group" aria-label="Filters">
        <select aria-label="Status" className={select} value={filters.status ?? ''} onChange={(e) => set('status', e.target.value)}>
          {statuses.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
        <select aria-label="Vessel" className={select} value={filters.vessel_id ?? ''} onChange={(e) => set('vessel_id', e.target.value)}>
          <option value="">All vessels</option>
          {options.vessels.map((v) => (
            <option key={v.id} value={v.id}>
              {v.name}
            </option>
          ))}
        </select>
        <select aria-label="Principal" className={select} value={filters.principal_id ?? ''} onChange={(e) => set('principal_id', e.target.value)}>
          <option value="">All principals</option>
          {principals.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
        <select aria-label="Rank" className={select} value={filters.rank_id ?? ''} onChange={(e) => set('rank_id', e.target.value)}>
          <option value="">All ranks</option>
          {options.ranks.map((r) => (
            <option key={r.id} value={r.id}>
              {r.name}
            </option>
          ))}
        </select>
        <div data-span="full" className="grid grid-cols-2 gap-2 max-sm:*:last:col-span-2 sm:contents">
          <FilterField label="Sign-on from">
            <input type="date" className={select} value={filters.sign_on_from ?? ''} onChange={(e) => set('sign_on_from', e.target.value)} />
          </FilterField>
          <FilterField label="Sign-on to" shortLabel="to">
            <input type="date" className={select} value={filters.sign_on_to ?? ''} onChange={(e) => set('sign_on_to', e.target.value)} />
          </FilterField>
          <FilterField label="Sign-off within (days)" shortLabel="Sign-off within" suffix="days">
            <input
              type="number"
              min={0}
              max={3650}
              className={`${select} sm:w-20!`}
              value={filters.sign_off_within_days ?? ''}
              onChange={(e) => set('sign_off_within_days', e.target.value)}
            />
          </FilterField>
        </div>
      </div>
      <DataGrid<Contract>
        id="contracts"
        columns={columns}
        filters={filters}
        searchPlaceholder="Search contract ID, crew or vessel…"
        rowKey={(r) => r.id}
        onRowClick={(r) => void navigate(`/contracts/${r.id}`)}
        defaultHidden={['created_at']}
        fetchPage={async (params, signal) => {
          const res = await contractsApi.list(params, signal)
          return { rows: res.data, total: res.meta.total }
        }}
      />
    </>
  )
}
