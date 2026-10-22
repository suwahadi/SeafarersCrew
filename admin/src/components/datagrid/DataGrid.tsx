import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { flexRender, getCoreRowModel, useReactTable, type ColumnDef, type SortingState, type VisibilityState } from '@tanstack/react-table'
import { useMemo, useState, type ReactNode } from 'react'
import { errorMessage, type Query } from '@/api/client'
import { useDebounced } from '@/hooks/useDebounced'
import { useLocalStorage } from '@/hooks/useLocalStorage'
import { Spinner } from '../ui/Spinner'
import { ColumnChooser } from './ColumnChooser'
import { Pagination, PAGE_SIZES } from './Pagination'

export interface GridGroup {
  key: string
  name: string
  count: number
}

export interface GridPage<T> {
  rows: T[]
  total: number
  groups?: GridGroup[]
}

export interface GridParams {
  page: number
  per_page: number
  sort?: string
  order?: 'asc' | 'desc'
  q?: string
}

export interface DataGridProps<T> {
  id: string
  columns: ColumnDef<T>[]
  fetchPage: (params: GridParams & Query, signal: AbortSignal) => Promise<GridPage<T>>
  filters?: Query
  toolbar?: ReactNode
  searchPlaceholder?: string
  rowKey: (row: T) => string
  onRowClick?: (row: T) => void
  groupKey?: (row: T) => string
  emptyMessage?: string
  defaultHidden?: string[]
}

export function DataGrid<T>({
  id,
  columns,
  fetchPage,
  filters = {},
  toolbar,
  searchPlaceholder = 'Search…',
  rowKey,
  onRowClick,
  groupKey,
  emptyMessage = 'No records found.',
  defaultHidden = [],
}: DataGridProps<T>) {
  const [search, setSearch] = useState('')
  const q = useDebounced(search.trim(), 300)
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useLocalStorage<number>(`grid:${id}:per_page`, 20)
  const [sorting, setSorting] = useState<SortingState>([])
  const [visibility, setVisibility] = useLocalStorage<VisibilityState>(
    `grid:${id}:columns`,
    Object.fromEntries(defaultHidden.map((c) => [c, false])),
  )

  const sort = sorting[0]
  const params: GridParams & Query = {
    ...filters,
    page,
    per_page: PAGE_SIZES.includes(perPage) ? perPage : 20,
    q,
    sort: sort?.id,
    order: sort ? (sort.desc ? 'desc' : 'asc') : undefined,
  }
  const filterKey = JSON.stringify(filters)
  const [lastKey, setLastKey] = useState(filterKey + q)
  if (lastKey !== filterKey + q) {
    setLastKey(filterKey + q)
    setPage(1)
  }

  const query = useQuery({
    queryKey: ['grid', id, params],
    queryFn: ({ signal }) => fetchPage(params, signal),
    placeholderData: keepPreviousData,
  })

  const rows = useMemo(() => query.data?.rows ?? [], [query.data])
  const table = useReactTable({
    data: rows,
    columns,
    state: { sorting, columnVisibility: visibility },
    onSortingChange: (updater) => {
      setSorting(typeof updater === 'function' ? updater(sorting) : updater)
      setPage(1)
    },
    onColumnVisibilityChange: (updater) => setVisibility(typeof updater === 'function' ? updater(visibility) : updater),
    getCoreRowModel: getCoreRowModel(),
    manualSorting: true,
    manualPagination: true,
    getRowId: rowKey,
  })

  const groups = query.data?.groups
  const groupNames = new Map(groups?.map((g) => [g.key, g]) ?? [])
  const visibleCount = table.getVisibleLeafColumns().length
  let lastGroup: string | undefined

  return (
    <div className="rounded-lg bg-white shadow-sm ring-1 ring-slate-200">
      <div className="flex flex-wrap items-center gap-2 border-b border-slate-200 p-3">
        <input
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={searchPlaceholder}
          aria-label="Search"
          className="h-9 w-full max-w-xs rounded-md border-0 px-3 text-sm ring-1 ring-inset ring-slate-300 focus:ring-2 focus:ring-navy-500"
        />
        {query.isFetching ? <Spinner size="sm" label="Updating" /> : null}
        <div className="ml-auto flex flex-wrap items-center gap-2">
          {toolbar}
          <ColumnChooser table={table} />
        </div>
      </div>
      {query.isError ? (
        <p role="alert" className="p-4 text-sm text-red-700">
          {errorMessage(query.error)}
        </p>
      ) : null}
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="bg-slate-50">
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((h) => {
                  const sorted = h.column.getIsSorted()
                  return (
                    <th
                      key={h.id}
                      scope="col"
                      aria-sort={sorted === 'asc' ? 'ascending' : sorted === 'desc' ? 'descending' : 'none'}
                      className="px-3 py-2 text-left text-xs font-semibold tracking-wide text-slate-600 uppercase whitespace-nowrap"
                    >
                      {h.isPlaceholder ? null : h.column.getCanSort() ? (
                        <button type="button" onClick={h.column.getToggleSortingHandler()} className="inline-flex items-center gap-1 tracking-wide uppercase hover:text-slate-900">
                          {flexRender(h.column.columnDef.header, h.getContext())}
                          <span aria-hidden className={sorted ? 'text-navy-700' : 'text-slate-400'}>
                            {sorted === 'asc' ? '▲' : sorted === 'desc' ? '▼' : '↕'}
                          </span>
                        </button>
                      ) : (
                        flexRender(h.column.columnDef.header, h.getContext())
                      )}
                    </th>
                  )
                })}
              </tr>
            ))}
          </thead>
          <tbody className="divide-y divide-slate-100">
            {query.isPending ? (
              <tr>
                <td colSpan={visibleCount} className="p-8 text-center">
                  <Spinner label="Loading records" />
                </td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={visibleCount} className="p-8 text-center text-sm text-slate-500">
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              table.getRowModel().rows.flatMap((row) => {
                const out: ReactNode[] = []
                const gk = groupKey?.(row.original)
                if (gk !== undefined && gk !== lastGroup) {
                  lastGroup = gk
                  const g = groupNames.get(gk)
                  out.push(
                    <tr key={`group-${gk}`} className="bg-navy-50">
                      <th colSpan={visibleCount} scope="rowgroup" className="px-3 py-1.5 text-left text-xs font-semibold text-navy-800">
                        {groupLabel(g)} <span className="font-normal text-slate-500">({g?.count ?? 0})</span>
                      </th>
                    </tr>,
                  )
                }
                out.push(
                  <tr
                    key={row.id}
                    onClick={onRowClick ? () => onRowClick(row.original) : undefined}
                    className={onRowClick ? 'cursor-pointer hover:bg-slate-50' : undefined}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} className="px-3 py-2 whitespace-nowrap text-slate-800">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>,
                )
                return out
              })
            )}
          </tbody>
        </table>
      </div>
      <Pagination
        page={page}
        perPage={params.per_page}
        total={query.data?.total ?? 0}
        onPage={setPage}
        onPerPage={(n) => {
          setPerPage(n)
          setPage(1)
        }}
      />
    </div>
  )
}

function groupLabel(g: GridGroup | undefined): string {
  if (g === undefined || g.name === '') return 'Not assigned'
  return g.name
}
