import type { ColumnDef } from '@tanstack/react-table'
import { useMemo, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'
import { errorMessage } from '@/api/client'
import { crewsApi, type CrewRow } from '@/api/crews'
import { departmentLabel } from '@/api/options'
import { DataGrid, type GridPage } from '@/components/datagrid/DataGrid'
import { Button } from '@/components/ui/Button'
import { ExpiryBadge, PageHeader } from '@/components/ui/Layout'
import { useToast } from '@/components/ui/Toast'
import { formatDate, formatSpan } from '@/lib/format'
import { ComplianceBadge, StatusBadge } from './badges'
import { CrewFilterPanel } from './CrewFilterPanel'
import { activeFilterCount, filtersFromParams, filtersToParams, filtersToQuery } from './filters'
import { useJobDownload } from './useJobDownload'

const columns: ColumnDef<CrewRow>[] = [
  {
    id: 'id',
    header: 'Crew ID',
    accessorKey: 'id',
    enableHiding: false,
    cell: ({ row }) => (
      <Link to={`/crews/${row.original.id}`} className="font-mono text-navy-700 hover:underline" onClick={(e) => e.stopPropagation()}>
        {row.original.id}
      </Link>
    ),
  },
  { id: 'full_name', header: 'Name', accessorKey: 'full_name', enableHiding: false },
  { id: 'rank', header: 'Rank', accessorKey: 'rank_name' },
  { id: 'department', header: 'Department', accessorFn: (r) => departmentLabel(r.department) },
  { id: 'status', header: 'Status', cell: ({ row }) => <StatusBadge status={row.original.status} /> },
  { id: 'nationality', header: 'Nationality', accessorKey: 'nationality' },
  { id: 'age', header: 'Age', accessorFn: (r) => r.age ?? '—' },
  { id: 'date_of_birth', header: 'Date of birth', accessorFn: (r) => formatDate(r.date_of_birth) },
  { id: 'mobile_phone', header: 'Mobile', accessorKey: 'mobile_phone' },
  { id: 'email', header: 'Email', accessorKey: 'email' },
  { id: 'passport_number', header: 'Passport No', accessorKey: 'passport_number', enableSorting: false },
  { id: 'seaman_book_number', header: 'Seaman Book No', accessorKey: 'seaman_book_number', enableSorting: false },
  { id: 'vessel', header: 'Vessel', accessorKey: 'vessel_name' },
  { id: 'principal', header: 'Principal', accessorKey: 'principal_name' },
  { id: 'compliant', header: 'Compliance', cell: ({ row }) => <ComplianceBadge compliant={row.original.compliant} /> },
  {
    id: 'nearest_expiry',
    header: 'Nearest expiry',
    cell: ({ row }) =>
      row.original.nearest_expiry ? (
        <span className="inline-flex items-center gap-2">
          {formatDate(row.original.nearest_expiry)}
          <ExpiryBadge badge={row.original.expiry_badge} />
        </span>
      ) : (
        '—'
      ),
  },
  { id: 'sea_service', header: 'Sea service', accessorFn: (r) => formatSpan(r.sea_service) },
]

export default function CrewListPage() {
  const [params, setParams] = useSearchParams()
  const navigate = useNavigate()
  const toast = useToast()
  const filters = useMemo(() => filtersFromParams(params), [params])
  const query = useMemo(() => filtersToQuery(filters), [filters])
  const [showFilters, setShowFilters] = useState(activeFilterCount(filters) > 0)
  const grouped = filters.group_by !== ''

  const exportCsv = useJobDownload(() => crewsApi.export('csv', query))
  const exportXlsx = useJobDownload(() => crewsApi.export('xlsx', query))
  const exporting = exportCsv.isPending || exportXlsx.isPending
  const runExport = (m: typeof exportCsv) =>
    m.mutate(undefined, {
      onSuccess: (job) => toast.success(`Export ready (${String((job.result as { rows?: number } | undefined)?.rows ?? 0)} rows).`),
      onError: (err) => toast.error(errorMessage(err)),
    })

  const count = activeFilterCount(filters)

  return (
    <>
      <PageHeader
        title="Crew"
        description="Seafarer records, documents, certificates and compliance."
        actions={
          <>
            <Button variant="secondary" onClick={() => void navigate('/crews/import')}>
              Import CSV
            </Button>
            <Button onClick={() => void navigate('/crews/new')}>New crew</Button>
          </>
        }
      />
      {showFilters ? <CrewFilterPanel key={params.toString()} value={filters} onApply={(f) => setParams(filtersToParams(f), { replace: true })} /> : null}
      <DataGrid<CrewRow>
        id="crews"
        columns={columns}
        filters={query}
        rowKey={(r) => r.id}
        searchPlaceholder="Name, Crew ID, passport or seaman book no."
        onRowClick={(r) => void navigate(`/crews/${r.id}`)}
        groupKey={grouped ? (r) => r.group_key ?? '' : undefined}
        defaultHidden={['date_of_birth', 'email', 'seaman_book_number', 'department']}
        fetchPage={async (p, signal): Promise<GridPage<CrewRow>> => {
          const res = await crewsApi.list(p, signal)
          return { rows: res.data.items, total: res.meta.total, groups: res.data.groups }
        }}
        toolbar={
          <>
            <Button variant="secondary" aria-expanded={showFilters} onClick={() => setShowFilters((s) => !s)}>
              Filters{count > 0 ? ` (${String(count)})` : ''}
            </Button>
            <Button variant="secondary" loading={exportCsv.isPending} disabled={exporting} onClick={() => runExport(exportCsv)}>
              Export CSV
            </Button>
            <Button variant="secondary" loading={exportXlsx.isPending} disabled={exporting} onClick={() => runExport(exportXlsx)}>
              Export Excel
            </Button>
          </>
        }
      />
    </>
  )
}
