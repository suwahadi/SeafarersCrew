import type { ColumnDef } from '@tanstack/react-table'
import { Link, useNavigate } from 'react-router'
import { vacanciesApi, type Vacancy } from '@/api/vacancies'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { Button } from '@/components/ui/Button'
import { inputClass } from '@/components/ui/Field'
import { Badge, PageHeader } from '@/components/ui/Layout'
import { useOptions } from '@/hooks/useOptions'
import { useUrlFilters } from '@/hooks/useUrlFilters'
import { formatDate, titleCase } from '@/lib/format'
import { DEPARTMENTS, VACANCY_STATUSES } from './vacancyForm'
import { VacancyStatusBadge } from './VacancyStatusBadge'

const FILTER_KEYS = ['status', 'rank_id', 'vessel_type_id', 'department'] as const

const columns: ColumnDef<Vacancy>[] = [
  {
    id: 'id',
    accessorFn: (r) => r.id,
    header: 'Vacancy ID',
    enableHiding: false,
    cell: ({ row }) => (
      <Link to={`/vacancies/${row.original.id}`} className="font-medium text-navy-700 hover:underline" onClick={(e) => e.stopPropagation()}>
        {row.original.id}
      </Link>
    ),
  },
  { id: 'title', accessorFn: (r) => r.title, header: 'Title' },
  { id: 'rank', accessorFn: (r) => r.rank.name, header: 'Rank' },
  { id: 'department', accessorFn: (r) => titleCase(r.department), header: 'Department' },
  { id: 'location', accessorFn: (r) => r.location || '—', header: 'Location', enableSorting: false },
  {
    id: 'closing_date',
    accessorFn: (r) => r.closing_date,
    header: 'Closing date',
    cell: ({ row }) => (
      <span className="inline-flex items-center gap-1">
        {formatDate(row.original.closing_date)}
        {row.original.is_expired ? <Badge tone="orange">Expired</Badge> : null}
      </span>
    ),
  },
  { id: 'status', accessorFn: (r) => r.status, header: 'Status', cell: ({ row }) => <VacancyStatusBadge vacancy={row.original} /> },
  {
    id: 'applications_count',
    accessorFn: (r) => r.applications_count,
    header: 'Applications',
    cell: ({ row }) => (
      <Link
        to={`/applicants?vacancy=${row.original.id}`}
        className="inline-flex items-center gap-1 text-navy-700 hover:underline"
        onClick={(e) => e.stopPropagation()}
        aria-label={`${row.original.applications_count} applications for ${row.original.title}`}
      >
        {row.original.applications_count}
        {row.original.new_applications_count > 0 ? <Badge tone="blue">{row.original.new_applications_count} new</Badge> : null}
      </Link>
    ),
  },
  { id: 'published_at', accessorFn: (r) => r.published_at, header: 'Published', cell: ({ row }) => formatDate(row.original.published_at) },
  { id: 'created_at', accessorFn: (r) => r.created_at, header: 'Created', cell: ({ row }) => formatDate(row.original.created_at) },
]

export default function VacancyListPage() {
  const navigate = useNavigate()
  const { options } = useOptions()
  const [filters, set] = useUrlFilters(FILTER_KEYS)
  const select = `h-9 ${inputClass} w-auto!`

  return (
    <>
      <PageHeader
        title="Vacancies"
        description="Job openings published on the career page."
        actions={<Button onClick={() => void navigate('/vacancies/new')}>New vacancy</Button>}
      />
      <div className="mb-3 flex flex-wrap items-center gap-2" role="group" aria-label="Filters">
        <select aria-label="Status" className={select} value={filters.status} onChange={(e) => set('status', e.target.value)}>
          <option value="">All statuses</option>
          {VACANCY_STATUSES.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
        <select aria-label="Rank" className={select} value={filters.rank_id} onChange={(e) => set('rank_id', e.target.value)}>
          <option value="">All ranks</option>
          {options.ranks.map((r) => (
            <option key={r.id} value={r.id}>
              {r.name}
            </option>
          ))}
        </select>
        <select aria-label="Vessel type" className={select} value={filters.vessel_type_id} onChange={(e) => set('vessel_type_id', e.target.value)}>
          <option value="">All vessel types</option>
          {options.vessel_types.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </select>
        <select aria-label="Department" className={select} value={filters.department} onChange={(e) => set('department', e.target.value)}>
          <option value="">All departments</option>
          {DEPARTMENTS.map((d) => (
            <option key={d.value} value={d.value}>
              {d.label}
            </option>
          ))}
        </select>
      </div>
      <DataGrid<Vacancy>
        id="vacancies"
        columns={columns}
        filters={filters}
        searchPlaceholder="Search title or vacancy ID…"
        rowKey={(r) => r.id}
        onRowClick={(r) => void navigate(`/vacancies/${r.id}`)}
        defaultHidden={['published_at', 'created_at']}
        emptyMessage="No vacancies yet."
        fetchPage={async (params, signal) => {
          const res = await vacanciesApi.list(params, signal)
          return { rows: res.data, total: res.meta.total }
        }}
      />
    </>
  )
}
