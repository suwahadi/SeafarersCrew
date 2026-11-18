import type { ColumnDef } from '@tanstack/react-table'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate } from 'react-router'
import { payrollApi, payrollKeys, type PayrollPeriod, type PayrollPeriodInput } from '@/api/payroll'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { PageHeader } from '@/components/ui/Layout'
import { useToast } from '@/components/ui/Toast'
import { formatDate, formatMoney } from '@/lib/format'
import { PeriodFormDialog } from './PeriodFormDialog'

const columns: ColumnDef<PayrollPeriod>[] = [
  { id: 'title', header: 'Title', accessorKey: 'title', enableSorting: false },
  {
    id: 'period',
    header: 'Period',
    enableSorting: false,
    accessorFn: (r) => `${formatDate(r.period_start)} – ${formatDate(r.period_end)}`,
  },
  {
    id: 'transfer_date',
    header: 'Transfer date',
    enableSorting: false,
    accessorFn: (r) => formatDate(r.transfer_date),
  },
  {
    id: 'line_count',
    header: 'Rows',
    accessorKey: 'line_count',
    enableSorting: false,
  },
  {
    id: 'total_usd',
    header: 'Total USD',
    enableSorting: false,
    accessorFn: (r) => formatMoney(r.total_usd, 'USD'),
  },
  {
    id: 'total_idr',
    header: 'Total IDR',
    enableSorting: false,
    accessorFn: (r) => formatMoney(r.total_idr, 'IDR', 0),
  },
]

export default function PayrollPeriodListPage() {
  const navigate = useNavigate()
  const toast = useToast()
  const qc = useQueryClient()
  const [creating, setCreating] = useState(false)

  const create = async (input: PayrollPeriodInput) => {
    const period = await payrollApi.createPeriod(input)
    await qc.invalidateQueries({ queryKey: payrollKeys.all })
    await qc.invalidateQueries({ queryKey: ['grid', 'payroll-periods'] })
    toast.success('Payroll period created.')
    void navigate(`/payroll/${period.id}`)
  }

  return (
    <>
      <PageHeader
        title="Payroll"
        description="Monthly salary transfer lists. Import the bank sheet, filter it and export it back."
        actions={
          <Button icon={<Icon name="plus" />} onClick={() => setCreating(true)}>
            New period
          </Button>
        }
      />
      <DataGrid<PayrollPeriod>
        id="payroll-periods"
        columns={columns}
        searchPlaceholder="Search period title…"
        rowKey={(r) => r.id}
        onRowClick={(r) => void navigate(`/payroll/${r.id}`)}
        emptyMessage="No payroll periods yet. Create one to import a salary sheet."
        fetchPage={async (params, signal) => {
          const res = await payrollApi.listPeriods(params, signal)
          return { rows: res.data, total: res.meta.total }
        }}
      />
      {creating ? <PeriodFormDialog onSubmit={create} onClose={() => setCreating(false)} /> : null}
    </>
  )
}
