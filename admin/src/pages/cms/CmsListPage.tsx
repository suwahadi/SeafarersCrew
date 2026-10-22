import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState, type ComponentType, type ReactNode } from 'react'
import { ApiError, errorMessage } from '@/api/client'
import { cmsApi, cmsKey, fetchAllCms, type CmsBase, type CmsSegment } from '@/api/cms'
import { Button } from '@/components/ui/Button'
import { DragHandle } from '@/components/ui/DragHandle'
import { Icon } from '@/components/ui/Icon'
import { EmptyState, PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog, Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Tabs } from '@/components/ui/Tabs'
import { useToast } from '@/components/ui/Toast'
import { useSortable } from '@/hooks/useSortable'
import { moveByIdWithinSubset } from '@/lib/reorder'

export interface CmsFormProps<T> {
  row: T | null
  onDone: () => void
}

export interface CmsTab<T> {
  id: string
  label: string
  match: (row: T) => boolean
}

export interface CmsConfig<T extends CmsBase> {
  segment: CmsSegment
  title: string
  singular: string
  description: string
  layout: 'list' | 'grid'
  compact?: boolean
  label: (row: T) => string
  summary: (row: T) => ReactNode
  toPayload: (row: T, isActive: boolean) => Record<string, unknown>
  Form: ComponentType<CmsFormProps<T>>
  lockedReason?: (row: T) => { active?: string; remove?: string } | null
  tabs?: CmsTab<T>[]
  notice?: ReactNode
  emptyHint?: string
}

interface CmsListPageProps<T extends CmsBase> {
  config: CmsConfig<T>
  headerActions?: ReactNode
}

export function CmsListPage<T extends CmsBase>({ config, headerActions }: CmsListPageProps<T>) {
  const qc = useQueryClient()
  const toast = useToast()
  const key = cmsKey(config.segment)
  const svc = useMemo(() => cmsApi<T>(config.segment), [config.segment])
  const rows = useQuery({ queryKey: key, queryFn: ({ signal }) => fetchAllCms<T>(config.segment, signal) })
  const [tab, setTab] = useState('all')
  const [editing, setEditing] = useState<T | null>(null)
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<T | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const all = useMemo(() => rows.data ?? [], [rows.data])
  const activeTab = config.tabs?.find((t) => t.id === tab)
  const visible = useMemo(() => (activeTab ? all.filter(activeTab.match) : all), [all, activeTab])
  const visibleIds = useMemo(() => new Set(visible.map((r) => r.id)), [visible])

  const reorder = useMutation({
    mutationFn: (next: T[]) => svc.reorder(next.map((r) => r.id)),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: key })
      const previous = qc.getQueryData<T[]>(key)
      qc.setQueryData<T[]>(key, next)
      return { previous }
    },
    onError: (err, _next, ctx) => {
      if (ctx?.previous) qc.setQueryData<T[]>(key, ctx.previous)
      toast.error(errorMessage(err))
    },
    onSettled: () => qc.invalidateQueries({ queryKey: key }),
  })

  const toggle = useMutation({
    mutationFn: (row: T) => svc.update(row.id, config.toPayload(row, !row.is_active)),
    onSuccess: (saved) => qc.setQueryData<T[]>(key, (cur) => cur?.map((r) => (r.id === saved.id ? saved : r))),
    onError: (err) => toast.error(errorMessage(err)),
  })

  const remove = useMutation({
    mutationFn: (row: T) => svc.remove(row.id),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: key })
      toast.success(`${config.singular} deleted.`)
      setDeleting(null)
    },
    onError: (err) => setDeleteError(err instanceof ApiError ? err.message : errorMessage(err)),
  })

  const sort = useSortable(
    visible.map((r) => r.id),
    (from, to) => reorder.mutate(moveByIdWithinSubset(all, visibleIds, from, to)),
  )

  const closeForm = () => {
    setCreating(false)
    setEditing(null)
  }

  const tabItems = config.tabs
    ? [{ id: 'all', label: `All (${all.length})` }, ...config.tabs.map((t) => ({ id: t.id, label: `${t.label} (${all.filter(t.match).length})` }))]
    : null

  const Form = config.Form
  const grid = config.layout === 'grid'
  const compact = config.compact === true

  return (
    <div>
      <PageHeader
        title={config.title}
        description={config.description}
        actions={
          <>
            {headerActions}
            <Button icon={<Icon name="plus" />} collapse onClick={() => setCreating(true)}>New {config.singular.toLowerCase()}</Button>
          </>
        }
      />
      {config.notice ? <div className="mb-4 text-sm text-slate-600">{config.notice}</div> : null}
      {tabItems ? (
        <div className="mb-4">
          <Tabs tabs={tabItems} active={tab} onChange={setTab} />
        </div>
      ) : null}
      {rows.isPending ? (
        <div className="flex justify-center p-8">
          <Spinner />
        </div>
      ) : rows.isError ? (
        <p role="alert" className="text-sm text-red-700">
          {errorMessage(rows.error)}
        </p>
      ) : visible.length === 0 ? (
        <EmptyState title={`No ${config.title.toLowerCase()} yet`}>{config.emptyHint}</EmptyState>
      ) : (
        <ol
          aria-label={config.title}
          className={grid ? 'grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4' : 'divide-y divide-slate-100 rounded-lg bg-white ring-1 ring-slate-200'}
        >
          {visible.map((row) => {
            const lock = config.lockedReason?.(row) ?? null
            const label = config.label(row)
            const handle = <DragHandle label={label} {...sort.handleProps(row.id)} />
            const controls = (
              <>
                <button
                  type="button"
                  role="switch"
                  aria-checked={row.is_active}
                  aria-label={`${label}: active`}
                  disabled={lock?.active !== undefined || toggle.isPending}
                  title={lock?.active}
                  onClick={() => toggle.mutate(row)}
                  className={`inline-flex h-6 w-10 shrink-0 items-center rounded-full p-0.5 transition-colors duration-150 ease-out disabled:cursor-not-allowed disabled:opacity-50 motion-reduce:transition-none ${row.is_active ? 'bg-green-600' : 'bg-slate-300'}`}
                >
                  <span className={`h-5 w-5 rounded-full bg-white shadow transition-transform duration-150 ease-out motion-reduce:transition-none ${row.is_active ? 'translate-x-4' : 'translate-x-0'}`} />
                </button>
                <span aria-hidden className={`hidden w-14 text-xs text-slate-600 ${compact ? '' : 'sm:inline'}`}>
                  {row.is_active ? 'Active' : 'Inactive'}
                </span>
              </>
            )
            const actions = (
              <>
                <Button size="sm" variant="ghost" icon={<Icon name="edit" />} collapse aria-label={`Edit ${label}`} onClick={() => setEditing(row)}>
                  Edit
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  icon={<Icon name="trash" />}
                  collapse
                  aria-label={`Delete ${label}`}
                  disabled={lock?.remove !== undefined}
                  title={lock?.remove}
                  onClick={() => {
                    setDeleteError(null)
                    setDeleting(row)
                  }}
                >
                  Delete
                </Button>
              </>
            )
            return (
              <li
                key={row.id}
                {...sort.rowProps(row.id)}
                className={`transition-opacity duration-150 ease-out motion-reduce:transition-none ${grid ? 'flex min-w-0 flex-col gap-3 rounded-lg bg-white p-3 shadow-sm ring-1 ring-slate-200' : compact ? 'flex items-center gap-2 p-2 sm:gap-3 sm:p-3' : 'flex flex-wrap items-center gap-x-2 gap-y-1 p-2 sm:gap-x-3 sm:p-3'}`}
              >
                {grid ? (
                  <>
                    <div className="min-w-0 flex-1">{config.summary(row)}</div>
                    <div className="flex items-center gap-1 border-t border-slate-100 pt-2">
                      {handle}
                      <div className="mr-auto flex items-center gap-2 pl-1">{controls}</div>
                      {actions}
                    </div>
                  </>
                ) : (
                  <>
                    {handle}
                    <div className={`min-w-0 flex-1 ${compact ? '' : 'basis-48 sm:basis-36'}`}>{config.summary(row)}</div>
                    <div className="ml-auto flex shrink-0 items-center gap-1">
                      <div className="mr-1 flex items-center gap-2">{controls}</div>
                      {actions}
                    </div>
                  </>
                )}
              </li>
            )
          })}
        </ol>
      )}
      <Modal open={creating || editing !== null} title={`${editing ? 'Edit' : 'New'} ${config.singular.toLowerCase()}`} onClose={closeForm} size="lg">
        {creating || editing ? <Form key={editing?.id ?? 'new'} row={editing} onDone={closeForm} /> : null}
      </Modal>
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
        message={
          deleteError ? (
            <p role="alert" className="text-red-700">
              {deleteError}
            </p>
          ) : (
            <p>Delete {deleting ? config.label(deleting) : config.singular.toLowerCase()}? This cannot be undone.</p>
          )
        }
      />
    </div>
  )
}
