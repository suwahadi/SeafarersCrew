import type { ColumnDef } from '@tanstack/react-table'
import { useNavigate } from 'react-router'
import { pagesApi, type PageSummary } from '@/api/pages'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { Button } from '@/components/ui/Button'
import { inputClass } from '@/components/ui/Field'
import { Badge, PageHeader } from '@/components/ui/Layout'
import { useUrlFilters } from '@/hooks/useUrlFilters'
import { formatDateTime } from '@/lib/format'

const FILTER_KEYS = ['status', 'is_system'] as const

const columns: ColumnDef<PageSummary>[] = [
  {
    id: 'title',
    accessorFn: (r) => r.title,
    header: 'Title',
    enableHiding: false,
    cell: ({ row }) => (
      <span className="inline-flex items-center gap-2 font-medium text-navy-700">
        {row.original.title}
        {row.original.is_system ? (
          <Badge tone="blue">
            <span aria-hidden>🔒</span>
            <span className="ml-1">System</span>
          </Badge>
        ) : null}
      </span>
    ),
  },
  { id: 'slug', accessorFn: (r) => r.path, header: 'Path', cell: ({ row }) => <code className="text-xs">{row.original.path}</code> },
  {
    id: 'status',
    accessorFn: (r) => r.status,
    header: 'Status',
    cell: ({ row }) => <Badge tone={row.original.status === 'published' ? 'green' : 'neutral'}>{row.original.status === 'published' ? 'Published' : 'Draft'}</Badge>,
  },
  {
    id: 'menus',
    accessorFn: (r) => `${r.in_header_menu ? 'header' : ''} ${r.in_footer_menu ? 'footer' : ''}`,
    header: 'Menus',
    enableSorting: false,
    cell: ({ row }) => (
      <span className="inline-flex gap-1">
        {row.original.in_header_menu ? <Badge tone="blue">Header</Badge> : null}
        {row.original.in_footer_menu ? <Badge tone="blue">Footer</Badge> : null}
        {!row.original.in_header_menu && !row.original.in_footer_menu ? '—' : null}
      </span>
    ),
  },
  { id: 'menu_order', accessorFn: (r) => r.menu_order, header: 'Menu order' },
  { id: 'updated_at', accessorFn: (r) => r.updated_at, header: 'Updated', cell: ({ row }) => formatDateTime(row.original.updated_at) },
]

export default function PageListPage() {
  const navigate = useNavigate()
  const [filters, set] = useUrlFilters(FILTER_KEYS)
  const select = `h-9 w-auto ${inputClass}`

  return (
    <>
      <PageHeader
        title="Pages"
        description="Website pages built from content blocks."
        actions={<Button onClick={() => void navigate('/website/pages/new')}>New page</Button>}
      />
      <div className="mb-3 flex flex-wrap items-center gap-2" role="group" aria-label="Filters">
        <select aria-label="Status" className={select} value={filters.status} onChange={(e) => set('status', e.target.value)}>
          <option value="">All statuses</option>
          <option value="draft">Draft</option>
          <option value="published">Published</option>
        </select>
        <select aria-label="Page kind" className={select} value={filters.is_system} onChange={(e) => set('is_system', e.target.value)}>
          <option value="">System and custom</option>
          <option value="true">System pages</option>
          <option value="false">Custom pages</option>
        </select>
      </div>
      <DataGrid<PageSummary>
        id="pages"
        columns={columns}
        filters={filters}
        searchPlaceholder="Search title or slug…"
        rowKey={(r) => r.id}
        onRowClick={(r) => void navigate(`/website/pages/${r.id}`)}
        defaultHidden={['menu_order']}
        emptyMessage="No pages found."
        fetchPage={async (params, signal) => {
          const res = await pagesApi.list(params, signal)
          return { rows: res.data, total: res.meta.total }
        }}
      />
    </>
  )
}
