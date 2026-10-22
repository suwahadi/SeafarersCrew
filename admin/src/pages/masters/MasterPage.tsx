import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { useMemo, useState } from 'react'
import { useParams } from 'react-router'
import { ApiError, errorMessage, type Query } from '@/api/client'
import { masterApi, optionsQuery, OPTIONS_KEY, type MasterRecord } from '@/api/masters'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { Badge, PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { isMasterSegment, masterConfigs, type MasterConfig } from './config'
import { MasterFormModal } from './MasterFormModal'
import { ReorderModal } from './ReorderModal'

export default function MasterPage() {
  const { master } = useParams()
  if (!isMasterSegment(master)) {
    return <p className="text-sm text-slate-600">Unknown master data list.</p>
  }
  return <MasterList key={master} config={masterConfigs[master]} />
}

function rowName(row: MasterRecord, config: MasterConfig): string {
  return typeof row.name === 'string' && row.name !== '' ? row.name : config.singular.toLowerCase()
}

export function MasterList({ config }: { config: MasterConfig }) {
  const qc = useQueryClient()
  const toast = useToast()
  const [editing, setEditing] = useState<MasterRecord | null>(null)
  const [creating, setCreating] = useState(false)
  const [reordering, setReordering] = useState(false)
  const [deleting, setDeleting] = useState<MasterRecord | null>(null)
  const [deleteError, setDeleteError] = useState<ApiError | null>(null)
  const [filters, setFilters] = useState<Record<string, string>>({ is_active: '' })
  const needsOptions = config.filters.some((f) => f.source)
  const options = useQuery({ ...optionsQuery, enabled: needsOptions })
  const svc = masterApi(config.segment)

  const invalidate = () => Promise.all([qc.invalidateQueries({ queryKey: ['grid', config.segment] }), qc.invalidateQueries({ queryKey: OPTIONS_KEY })])

  const toggle = useMutation({
    mutationFn: (row: MasterRecord) => svc.update(row.id, { ...row, is_active: !row.is_active }),
    onSuccess: invalidate,
    onError: (err) => toast.error(errorMessage(err)),
  })

  const remove = useMutation({
    mutationFn: (row: MasterRecord) => svc.remove(row.id),
    onSuccess: async () => {
      await invalidate()
      toast.success(`${config.singular} deleted.`)
      setDeleting(null)
    },
    onError: (err) => setDeleteError(err instanceof ApiError ? err : new ApiError(0, { code: 'error', message: errorMessage(err) })),
  })

  const columns = useMemo<ColumnDef<MasterRecord>[]>(
    () => [
      ...config.columns.map<ColumnDef<MasterRecord>>((c) => ({
        id: c.id,
        header: c.header,
        enableSorting: c.sortable,
        cell: ({ row }) => c.render(row.original),
      })),
      {
        id: 'is_active',
        header: 'Active',
        cell: ({ row }) => (
          <button
            type="button"
            role="switch"
            aria-checked={row.original.is_active}
            aria-label={`Active: ${row.original.is_active ? 'on' : 'off'}`}
            onClick={() => toggle.mutate(row.original)}
            className="rounded"
          >
            <Badge tone={row.original.is_active ? 'green' : 'neutral'}>{row.original.is_active ? 'Active' : 'Inactive'}</Badge>
          </button>
        ),
      },
      {
        id: 'actions',
        header: '',
        enableSorting: false,
        enableHiding: false,
        cell: ({ row }) => (
          <div className="flex justify-end gap-1">
            <Button size="sm" variant="ghost" icon={<Icon name="edit" />} collapse aria-label={`Edit ${rowName(row.original, config)}`} onClick={() => setEditing(row.original)}>
              Edit
            </Button>
            {config.readonlyOnSystem && row.original.is_system === true ? null : (
              <Button
                size="sm"
                variant="ghost"
                icon={<Icon name="trash" />}
                collapse
                aria-label={`Delete ${rowName(row.original, config)}`}
                onClick={() => {
                  setDeleteError(null)
                  setDeleting(row.original)
                }}
              >
                Delete
              </Button>
            )}
          </div>
        ),
      },
    ],
    [config, toggle],
  )

  const gridFilters: Query = Object.fromEntries(Object.entries(filters).filter(([, v]) => v !== ''))

  return (
    <div>
      <PageHeader
        title={config.title}
        description={config.description}
        actions={
          <>
            <Button variant="secondary" icon={<Icon name="grip" />} collapse onClick={() => setReordering(true)}>
              Reorder
            </Button>
            <Button icon={<Icon name="plus" />} collapse onClick={() => setCreating(true)}>New {config.singular.toLowerCase()}</Button>
          </>
        }
      />
      <DataGrid<MasterRecord>
        id={config.segment}
        columns={columns}
        filters={gridFilters}
        rowKey={(r) => r.id}
        searchPlaceholder={`Search ${config.title.toLowerCase()}…`}
        fetchPage={async (params, signal) => {
          const res = await svc.list(params, signal)
          return { rows: res.data, total: res.meta.total }
        }}
        toolbar={
          <div className="filter-bar min-w-0 flex-1 sm:flex-none sm:flex-nowrap">
            <FilterSelect label="Visibility" allLabel="Active and inactive" value={filters.is_active ?? ''} onChange={(v) => setFilters((f) => ({ ...f, is_active: v }))} options={[
              { value: 'true', label: 'Active' },
              { value: 'false', label: 'Inactive' },
            ]} />
            {config.filters.map((f) => (
              <FilterSelect
                key={f.name}
                label={f.label}
                value={filters[f.name] ?? ''}
                onChange={(v) => setFilters((cur) => ({ ...cur, [f.name]: v }))}
                options={f.options ?? (options.data && f.source ? f.source(options.data) : [])}
              />
            ))}
          </div>
        }
      />
      <MasterFormModal config={config} row={editing} open={creating || editing !== null} onClose={() => {
        setCreating(false)
        setEditing(null)
      }} />
      <ReorderModal config={config} open={reordering} onClose={() => setReordering(false)} />
      <ConfirmDialog
        open={deleting !== null}
        title={`Delete ${config.singular.toLowerCase()}`}
        danger
        confirmLabel="Delete"
        loading={remove.isPending}
        onClose={() => setDeleting(null)}
        onConfirm={() => {
          if (deleting) remove.mutate(deleting)
        }}
        message={<DeleteMessage config={config} row={deleting} error={deleteError} />}
      />
    </div>
  )
}

function FilterSelect({ label, allLabel, value, onChange, options }: { label: string; allLabel?: string; value: string; onChange: (v: string) => void; options: { value: string; label: string }[] }) {
  return (
    <label className="block text-sm text-slate-600">
      <span className="sr-only">{label}</span>
      <select value={value} onChange={(e) => onChange(e.target.value)} aria-label={label} className="h-9 w-full rounded-md border-0 bg-white text-sm text-slate-900 ring-1 ring-inset ring-slate-300 focus:ring-2 focus:ring-navy-500">
        <option value="">{allLabel ?? `All ${label.toLowerCase()}`}</option>
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  )
}

export function DeleteMessage({ config, row, error }: { config: MasterConfig; row: MasterRecord | null; error: ApiError | null }) {
  const name = row && typeof row.name === 'string' ? row.name : config.singular.toLowerCase()
  if (!error) return <p>Delete {name}? It will be hidden from lists and dropdowns.</p>
  if (error.code === 'in_use') {
    return (
      <div role="alert" className="space-y-2">
        <p className="font-medium text-red-700">{name} is still in use and cannot be deleted.</p>
        <ul className="list-disc pl-5">
          {Object.entries(error.fields).map(([label, count]) => (
            <li key={label}>
              {count} {label}
            </li>
          ))}
        </ul>
        <p>Deactivate it instead to hide it from new records.</p>
      </div>
    )
  }
  return (
    <p role="alert" className="text-red-700">
      {error.message}
    </p>
  )
}
