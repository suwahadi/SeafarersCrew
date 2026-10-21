import type { ColumnDef } from '@tanstack/react-table'
import { useNavigate } from 'react-router'
import { newsApi, type NewsSummary } from '@/api/news'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { Button } from '@/components/ui/Button'
import { inputClass } from '@/components/ui/Field'
import { PageHeader } from '@/components/ui/Layout'
import { useUrlFilters } from '@/hooks/useUrlFilters'
import { formatDateTime } from '@/lib/format'
import { NewsStatusBadge } from './NewsStatusBadge'

const FILTER_KEYS = ['status'] as const

const columns: ColumnDef<NewsSummary>[] = [
  {
    id: 'thumbnail',
    accessorFn: (r) => r.thumbnail_file_id,
    header: 'Thumbnail',
    enableSorting: false,
    cell: ({ row }) => {
      const image = row.original.thumbnail
      return (
        <span className="flex h-10 w-16 items-center justify-center overflow-hidden rounded bg-slate-100">
          {image ? <img src={image.thumb_url || image.url} alt="" width={64} height={40} className="h-full w-full object-cover" /> : null}
        </span>
      )
    },
  },
  { id: 'title', accessorFn: (r) => r.title, header: 'Title', enableHiding: false, cell: ({ row }) => <span className="font-medium text-navy-700">{row.original.title}</span> },
  { id: 'status', accessorFn: (r) => r.status, header: 'Status', cell: ({ row }) => <NewsStatusBadge news={row.original} /> },
  { id: 'published_at', accessorFn: (r) => r.published_at, header: 'Published', cell: ({ row }) => formatDateTime(row.original.published_at) },
  { id: 'updated_at', accessorFn: (r) => r.updated_at, header: 'Updated', cell: ({ row }) => formatDateTime(row.original.updated_at) },
]

export default function NewsListPage() {
  const navigate = useNavigate()
  const [filters, set] = useUrlFilters(FILTER_KEYS)

  return (
    <>
      <PageHeader title="News" description="Articles shown on the news page." actions={<Button onClick={() => void navigate('/website/news/new')}>New article</Button>} />
      <div className="mb-3 flex flex-wrap items-center gap-2" role="group" aria-label="Filters">
        <select aria-label="Status" className={`h-9 w-auto ${inputClass}`} value={filters.status} onChange={(e) => set('status', e.target.value)}>
          <option value="">All statuses</option>
          <option value="draft">Draft</option>
          <option value="published">Published</option>
        </select>
      </div>
      <DataGrid<NewsSummary>
        id="news"
        columns={columns}
        filters={filters}
        searchPlaceholder="Search title, slug or excerpt…"
        rowKey={(r) => r.id}
        onRowClick={(r) => void navigate(`/website/news/${r.id}`)}
        emptyMessage="No articles yet."
        fetchPage={async (params, signal) => {
          const res = await newsApi.list(params, signal)
          return { rows: res.data, total: res.meta.total }
        }}
      />
    </>
  )
}
