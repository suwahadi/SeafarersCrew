import type { ColumnDef } from '@tanstack/react-table'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useRef, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router'
import { errorMessage, type Query } from '@/api/client'
import { payrollApi, payrollKeys, type PayrollLine, type PayrollPeriodInput } from '@/api/payroll'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { formatDate, formatMoney } from '@/lib/format'
import { useJobDownload } from '@/pages/crews/useJobDownload'
import { activeFilterCount, filtersFromParams, filtersToParams, filtersToQuery, toSearchParams } from './filters'
import { LineFormDialog } from './LineFormDialog'
import { PayrollFilterPanel } from './PayrollFilterPanel'
import { PayrollImportDialog } from './PayrollImportDialog'
import { PayrollTotals } from './PayrollTotals'
import { PeriodFormDialog } from './PeriodFormDialog'

const number = (text: string) => <span className="block text-right tabular-nums">{text}</span>

const columns: ColumnDef<PayrollLine>[] = [
  {
    id: 'line_no',
    header: 'No',
    accessorKey: 'line_no',
    cell: ({ row }) => number(String(row.original.line_no)),
  },
  {
    id: 'crew_name',
    header: 'Name',
    accessorKey: 'crew_name',
    cell: ({ row }) => <span className="font-medium text-navy-800">{row.original.crew_name}</span>,
  },
  {
    id: 'employee_no',
    header: 'Employee ID',
    accessorKey: 'employee_no',
  },
  {
    id: 'rank_code',
    header: 'Rank',
    accessorKey: 'rank_code',
  },
  {
    id: 'vessel_name',
    header: 'Vessel',
    accessorKey: 'vessel_name',
    cell: ({ row }) => <span className="whitespace-nowrap">{row.original.vessel_name}</span>,
  },
  {
    id: 'bank_name',
    header: 'Bank',
    accessorKey: 'bank_name',
    cell: ({ row }) => <span className="whitespace-nowrap">{row.original.bank_name}</span>,
  },
  {
    id: 'account_name',
    header: 'Account name',
    accessorKey: 'account_name',
    enableSorting: false,
  },
  {
    id: 'account_number',
    header: 'Account no.',
    enableSorting: false,
    cell: ({ row }) => <span className="font-mono whitespace-nowrap">{row.original.account_number}</span>,
  },
  {
    id: 'amount_usd',
    header: 'USD',
    accessorKey: 'amount_usd',
    cell: ({ row }) => number(row.original.amount_usd === null ? '—' : formatMoney(row.original.amount_usd, 'USD')),
  },
  {
    id: 'amount_idr',
    header: 'IDR',
    accessorKey: 'amount_idr',
    cell: ({ row }) => number(formatMoney(row.original.amount_idr, 'IDR', 0)),
  },
  {
    id: 'transfer_date',
    header: 'Date',
    accessorKey: 'transfer_date',
    cell: ({ row }) => <span className="whitespace-nowrap">{formatDate(row.original.transfer_date)}</span>,
  },
  {
    id: 'remark',
    header: 'Remark',
    accessorKey: 'remark',
    enableSorting: false,
    cell: ({ row }) => <span className="block min-w-48 whitespace-normal">{row.original.remark}</span>,
  },
]

export default function PayrollPeriodPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const [params, setParams] = useSearchParams()
  const filters = useMemo(() => filtersFromParams(params), [params])
  const count = activeFilterCount(filters)
  const [showFilters, setShowFilters] = useState(count > 0)
  const [editing, setEditing] = useState<PayrollLine | 'new' | null>(null)
  const [editingPeriod, setEditingPeriod] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [importing, setImporting] = useState(false)
  const lastQuery = useRef<Query>({})

  const period = useQuery({
    queryKey: payrollKeys.period(id),
    queryFn: () => payrollApi.getPeriod(id),
  })
  const facets = useQuery({
    queryKey: payrollKeys.facets(id),
    queryFn: () => payrollApi.facets(id),
  })
  const exportXlsx = useJobDownload(() => payrollApi.export(id, toSearchParams(lastQuery.current)))
  const removePeriod = useMutation({
    mutationFn: () => payrollApi.deletePeriod(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['grid', 'payroll-periods'] })
      toast.success('Payroll period deleted.')
      void navigate('/payroll')
    },
    onError: (err) => toast.error(errorMessage(err)),
  })

  const updatePeriod = async (input: PayrollPeriodInput) => {
    await payrollApi.updatePeriod(id, input)
    await qc.invalidateQueries({ queryKey: payrollKeys.period(id) })
    void qc.invalidateQueries({ queryKey: ['grid', 'payroll-periods'] })
    toast.success('Payroll period saved.')
    setEditingPeriod(false)
  }

  if (period.isPending) {
    return (
      <div className="flex justify-center p-12 text-navy-800">
        <Spinner size="lg" label="Loading payroll period" />
      </div>
    )
  }
  if (period.isError)
    return (
      <p role="alert" className="text-sm text-red-700">
        {errorMessage(period.error)}
      </p>
    )
  const p = period.data

  return (
    <>
      <PageHeader
        title={p.title}
        description={`${formatDate(p.period_start)} – ${formatDate(p.period_end)} · Transfer date ${p.transfer_date ? formatDate(p.transfer_date) : 'not set'}`}
        actions={
          <>
            <Button variant="ghost" collapse icon={<Icon name="edit" />} onClick={() => setEditingPeriod(true)}>
              Edit period
            </Button>
            <Button variant="ghost" collapse icon={<Icon name="trash" />} onClick={() => setDeleting(true)}>
              Delete period
            </Button>
            <Button variant="secondary" collapse icon={<Icon name="upload" />} onClick={() => setImporting(true)}>
              Import xlsx
            </Button>
            <Button
              variant="secondary"
              collapse
              icon={<Icon name="download" />}

              loading={exportXlsx.isPending}
              onClick={() =>
                exportXlsx.mutate(undefined, {
                  onError: (err) => toast.error(errorMessage(err)),
                })
              }
            >
              Export xlsx
            </Button>
            <Button collapse icon={<Icon name="plus" />} onClick={() => setEditing('new')}>
              Add row
            </Button>
          </>
        }
      />
      {showFilters ? (
        <PayrollFilterPanel
          key={params.toString()}
          value={filters}
          facets={facets.data ?? { banks: [], vessels: [], ranks: [] }}
          onApply={(f) => setParams(filtersToParams(f), { replace: true })}
        />
      ) : null}
      <DataGrid<PayrollLine>
        id="payroll-lines"
        columns={columns}
        filters={filtersToQuery(filters)}
        rowKey={(r) => r.id}
        searchPlaceholder="Name, Employee ID, account no. or remark"
        emptyMessage={count > 0 ? 'No rows match these filters.' : 'No rows yet. Import the salary sheet or add a row.'}
        onRowClick={(r) => setEditing(r)}
        fetchPage={async (query, signal) => {
          lastQuery.current = query
          const res = await payrollApi.lines(id, toSearchParams(query), signal)
          return {
            rows: res.data,
            total: res.meta.total,
            summary: <PayrollTotals totals={res.totals} />,
          }
        }}
        toolbar={
          <Button
            variant="secondary"
            collapse
            aria-expanded={showFilters}

            icon={
              <span className="relative inline-flex">
                <Icon name="filter" />
                {count > 0 ? (
                  <span aria-hidden="true" className="absolute -top-2 -right-2 min-w-4 rounded-full bg-navy-800 px-1 text-center text-[10px] leading-4 text-white">
                    {count}
                  </span>
                ) : null}
              </span>
            }
            onClick={() => setShowFilters((v) => !v)}
          >
            Filter{count > 0 ? ` (${count})` : ''}
          </Button>
        }
      />
      {editing ? <LineFormDialog periodId={id} line={editing === 'new' ? undefined : editing} onClose={() => setEditing(null)} /> : null}
      {importing ? <PayrollImportDialog periodId={id} onClose={() => setImporting(false)} /> : null}
      {editingPeriod ? <PeriodFormDialog period={p} onSubmit={updatePeriod} onClose={() => setEditingPeriod(false)} /> : null}
      <ConfirmDialog
        open={deleting}
        title="Delete this payroll period?"
        message={`${p.title} and its ${p.line_count} rows are removed. Bank accounts added to crew profiles stay. This cannot be undone.`}
        confirmLabel="Delete"
        danger
        loading={removePeriod.isPending}
        onConfirm={() => removePeriod.mutate()}
        onClose={() => setDeleting(false)}
      />
    </>
  )
}
