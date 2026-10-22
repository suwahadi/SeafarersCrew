import type { ColumnDef } from '@tanstack/react-table'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { quotationsApi, type Quotation } from '@/api/quotations'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { FilterField, filterClass } from '@/components/ui/Field'
import { PageHeader } from '@/components/ui/Layout'
import { useOptions } from '@/hooks/useOptions'
import { formatDate, formatMoney } from '@/lib/format'
import { StatusBadge } from '../contracts/ContractStatusBadge'

const columns: ColumnDef<Quotation>[] = [
  {
    id: 'id',
    accessorFn: (r) => r.id,
    header: 'Quotation ID',
    enableHiding: false,
    cell: ({ row }) => (
      <Link to={`/quotations/${row.original.id}`} className="font-medium text-navy-700 hover:underline" onClick={(e) => e.stopPropagation()}>
        {row.original.id}
      </Link>
    ),
  },
  { id: 'client', accessorFn: (r) => r.client_name, header: 'Client' },
  { id: 'vessel', accessorFn: (r) => r.vessel_name || '—', header: 'Vessel' },
  { id: 'quotation_date', accessorFn: (r) => formatDate(r.quotation_date), header: 'Date' },
  { id: 'valid_until', accessorFn: (r) => formatDate(r.valid_until), header: 'Valid until' },
  { id: 'currency', accessorFn: (r) => r.currency_code, header: 'Currency' },
  { id: 'grand_total', accessorFn: (r) => formatMoney(r.grand_total, r.currency_code, r.currency_decimals), header: 'Grand total' },
  { id: 'status', accessorFn: (r) => r.status, header: 'Status', cell: ({ row }) => <StatusBadge status={row.original.status} /> },
]

const statuses = ['draft', 'sent', 'accepted', 'rejected', 'expired']

export default function QuotationListPage() {
  const navigate = useNavigate()
  const { options } = useOptions()
  const [filters, setFilters] = useState<Record<string, string>>({})
  const set = (key: string, value: string) => setFilters((f) => ({ ...f, [key]: value }))
  const select = filterClass

  return (
    <>
      <PageHeader
        title="Quotations"
        description="Crew supply quotations with daily rates and lump sums."
        actions={
          <Button icon={<Icon name="plus" />} onClick={() => void navigate('/quotations/new')}>
            New quotation
          </Button>
        }
      />
      <div className="filter-bar mb-3" role="group" aria-label="Filters">
        <select aria-label="Status" className={select} value={filters.status ?? ''} onChange={(e) => set('status', e.target.value)}>
          <option value="">All statuses</option>
          {statuses.map((s) => (
            <option key={s} value={s}>
              {s.charAt(0).toUpperCase() + s.slice(1)}
            </option>
          ))}
        </select>
        <select aria-label="Client" className={select} value={filters.client_company_id ?? ''} onChange={(e) => set('client_company_id', e.target.value)}>
          <option value="">All clients</option>
          {options.companies.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
        <select aria-label="Currency" className={select} value={filters.currency ?? ''} onChange={(e) => set('currency', e.target.value)}>
          <option value="">All currencies</option>
          {options.currencies.map((c) => (
            <option key={c.code} value={c.code}>
              {c.code}
            </option>
          ))}
        </select>
        <FilterField label="From">
          <input type="date" className={select} value={filters.date_from ?? ''} onChange={(e) => set('date_from', e.target.value)} />
        </FilterField>
        <FilterField label="To" shortLabel="to">
          <input type="date" className={select} value={filters.date_to ?? ''} onChange={(e) => set('date_to', e.target.value)} />
        </FilterField>
      </div>
      <DataGrid<Quotation>
        id="quotations"
        columns={columns}
        filters={filters}
        searchPlaceholder="Search quotation ID, client or vessel…"
        rowKey={(r) => r.id}
        onRowClick={(r) => void navigate(`/quotations/${r.id}`)}
        fetchPage={async (params, signal) => {
          const res = await quotationsApi.list(params, signal)
          return { rows: res.data, total: res.meta.total }
        }}
      />
    </>
  )
}
