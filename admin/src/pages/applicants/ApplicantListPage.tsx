import { useQuery } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { Link, useNavigate, useSearchParams } from 'react-router'
import { applicantsApi, type Applicant } from '@/api/applicants'
import { vacanciesApi } from '@/api/vacancies'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { MultiSelect } from '@/components/forms/MultiSelect'
import { Button } from '@/components/ui/Button'
import { inputClass } from '@/components/ui/Field'
import { PageHeader } from '@/components/ui/Layout'
import { useDebounced } from '@/hooks/useDebounced'
import { useOptions } from '@/hooks/useOptions'
import { useUrlFilters } from '@/hooks/useUrlFilters'
import { formatDateTime } from '@/lib/format'
import { STAGES, STAGE_LABELS } from '@/lib/pipeline'
import { ApplicantKanban } from './ApplicantKanban'
import { StageBadge } from './StageBadge'

const FILTER_KEYS = ['vacancy', 'stage', 'rank_id', 'date_from', 'date_to', 'q'] as const

const columns: ColumnDef<Applicant>[] = [
  {
    id: 'id',
    accessorFn: (r) => r.id,
    header: 'Application ID',
    enableHiding: false,
    cell: ({ row }) => (
      <Link to={`/applicants/${row.original.id}`} className="font-medium text-navy-700 hover:underline" onClick={(e) => e.stopPropagation()}>
        {row.original.id}
      </Link>
    ),
  },
  { id: 'full_name', accessorFn: (r) => r.full_name, header: 'Name' },
  { id: 'vacancy', accessorFn: (r) => r.vacancy.title, header: 'Vacancy' },
  { id: 'rank', accessorFn: (r) => r.rank_applied?.name ?? '—', header: 'Rank applied' },
  { id: 'years_experience', accessorFn: (r) => r.years_experience, header: 'Years', cell: ({ row }) => row.original.years_experience ?? '—' },
  { id: 'nationality', accessorFn: (r) => r.nationality || '—', header: 'Nationality', enableSorting: false },
  { id: 'email', accessorFn: (r) => r.email, header: 'Email', enableSorting: false },
  { id: 'stage', accessorFn: (r) => r.stage, header: 'Stage', cell: ({ row }) => <StageBadge stage={row.original.stage} /> },
  { id: 'files_count', accessorFn: (r) => r.files_count, header: 'Files', enableSorting: false },
  { id: 'created_at', accessorFn: (r) => r.created_at, header: 'Applied', cell: ({ row }) => formatDateTime(row.original.created_at) },
]

const split = (v: string) => (v === '' ? [] : v.split(','))

export default function ApplicantListPage() {
  const navigate = useNavigate()
  const { options } = useOptions()
  const [params, setParams] = useSearchParams()
  const [filters, set] = useUrlFilters(FILTER_KEYS)
  const kanban = params.get('view') === 'kanban'
  const select = `h-9 ${inputClass} w-auto!`

  const vacancies = useQuery({
    queryKey: ['vacancy-options'],
    queryFn: ({ signal }) => vacanciesApi.list({ per_page: 100, sort: 'title', order: 'asc' }, signal),
  })
  const vacancyOptions = vacancies.data?.data ?? []
  const selectedMissing = filters.vacancy !== '' && !vacancyOptions.some((v) => v.id === filters.vacancy)

  const setView = (view: 'list' | 'kanban') => {
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        if (view === 'kanban') next.set('view', 'kanban')
        else next.delete('view')
        return next
      },
      { replace: true },
    )
  }

  const kanbanSearch = useDebounced(filters.q.trim(), 300)
  const gridFilters = { vacancy: filters.vacancy, stage: filters.stage, rank_id: filters.rank_id, date_from: filters.date_from, date_to: filters.date_to }
  const kanbanFilters = { vacancy: filters.vacancy, rank_id: filters.rank_id, date_from: filters.date_from, date_to: filters.date_to, q: kanbanSearch }

  return (
    <>
      <PageHeader
        title="Applicants"
        description="Applications received through the career page."
        actions={
          <div role="group" aria-label="View" className="inline-flex overflow-hidden rounded-md ring-1 ring-slate-300">
            <Button variant={kanban ? 'secondary' : 'primary'} size="sm" className="rounded-none" aria-pressed={!kanban} onClick={() => setView('list')}>
              List
            </Button>
            <Button variant={kanban ? 'primary' : 'secondary'} size="sm" className="rounded-none" aria-pressed={kanban} onClick={() => setView('kanban')}>
              Kanban
            </Button>
          </div>
        }
      />
      <div className="mb-3 flex flex-wrap items-center gap-2" role="group" aria-label="Filters">
        <select aria-label="Vacancy" className={`${select} max-w-64`} value={filters.vacancy} onChange={(e) => set('vacancy', e.target.value)}>
          <option value="">All vacancies</option>
          {selectedMissing ? <option value={filters.vacancy}>{filters.vacancy}</option> : null}
          {vacancyOptions.map((v) => (
            <option key={v.id} value={v.id}>
              {v.title} ({v.id})
            </option>
          ))}
        </select>
        {kanban ? null : (
          <MultiSelect
            label="Stage"
            hideLabel
            placeholder="All stages"
            options={STAGES.map((s) => ({ value: s, label: STAGE_LABELS[s] }))}
            value={split(filters.stage)}
            onChange={(v) => set('stage', v.join(','))}
          />
        )}
        <MultiSelect
          label="Rank"
          hideLabel
          placeholder="All ranks"
          options={options.ranks.map((r) => ({ value: r.id, label: r.name }))}
          value={split(filters.rank_id)}
          onChange={(v) => set('rank_id', v.join(','))}
        />
        <label className="flex items-center gap-1 text-sm text-slate-600">
          Applied from
          <input type="date" className={select} value={filters.date_from} onChange={(e) => set('date_from', e.target.value)} />
        </label>
        <label className="flex items-center gap-1 text-sm text-slate-600">
          to
          <input type="date" className={select} value={filters.date_to} onChange={(e) => set('date_to', e.target.value)} />
        </label>
        {kanban ? (
          <input
            type="search"
            aria-label="Search applicants"
            placeholder="Search name, email or application ID…"
            value={filters.q}
            onChange={(e) => set('q', e.target.value)}
            className={`${select} w-72`}
          />
        ) : null}
      </div>
      {kanban ? (
        <ApplicantKanban filters={kanbanFilters} />
      ) : (
        <DataGrid<Applicant>
          id="applicants"
          columns={columns}
          filters={gridFilters}
          searchPlaceholder="Search name, email or application ID…"
          rowKey={(r) => r.id}
          onRowClick={(r) => void navigate(`/applicants/${r.id}`)}
          defaultHidden={['nationality', 'email', 'files_count']}
          emptyMessage="No applicants yet."
          fetchPage={async (params2, signal) => {
            const res = await applicantsApi.list(params2, signal)
            return { rows: res.data, total: res.meta.total }
          }}
        />
      )}
    </>
  )
}
