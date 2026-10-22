import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage } from '@/api/client'
import { fetchAllMasterRows, masterApi, OPTIONS_KEY, type MasterRecord } from '@/api/masters'
import { Button } from '@/components/ui/Button'
import { DragHandle } from '@/components/ui/DragHandle'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { useSortable } from '@/hooks/useSortable'
import type { MasterConfig } from './config'
import { moveItem } from './reorder'

export function ReorderModal({ config, open, onClose }: { config: MasterConfig; open: boolean; onClose: () => void }) {
  return (
    <Modal open={open} title={`Reorder ${config.title.toLowerCase()}`} onClose={onClose} size="md">
      {open ? <ReorderList config={config} onDone={onClose} /> : null}
    </Modal>
  )
}

function rowLabel(row: MasterRecord): string {
  const name = typeof row.name === 'string' ? row.name : ''
  const code = typeof row.code === 'string' ? row.code : typeof row.short_code === 'string' ? row.short_code : ''
  return code && name ? `${name} (${code})` : name || code
}

export function ReorderList({ config, onDone }: { config: MasterConfig; onDone: () => void }) {
  const qc = useQueryClient()
  const toast = useToast()
  const rows = useQuery({ queryKey: ['reorder', config.segment], queryFn: () => fetchAllMasterRows(config.segment), gcTime: 0 })
  const [order, setOrder] = useState<MasterRecord[] | null>(null)
  const items = order ?? rows.data ?? []

  const save = useMutation({
    mutationFn: () => masterApi(config.segment).reorder(items.map((r) => r.id)),
    onSuccess: async () => {
      await Promise.all([qc.invalidateQueries({ queryKey: ['grid', config.segment] }), qc.invalidateQueries({ queryKey: OPTIONS_KEY })])
      toast.success('Order saved.')
      onDone()
    },
    onError: (err) => toast.error(errorMessage(err)),
  })

  const sort = useSortable(
    items.map((r) => r.id),
    (from, to) =>
      setOrder(
        moveItem(
          items,
          items.findIndex((r) => r.id === from),
          items.findIndex((r) => r.id === to),
        ),
      ),
  )

  if (rows.isPending) {
    return (
      <div className="flex justify-center p-6">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-600">Drag a row by its handle, or focus the handle and press the up and down arrow keys, to set the order used in every dropdown.</p>
      <ol className="divide-y divide-slate-100 rounded-md ring-1 ring-slate-200" aria-label="Order">
        {items.map((row) => (
          <li key={row.id} {...sort.rowProps(row.id)} className="flex items-center gap-2 bg-white px-2 py-1 text-sm">
            <DragHandle label={rowLabel(row)} {...sort.handleProps(row.id)} />
            <span className="min-w-0 flex-1 break-words">{rowLabel(row)}</span>
          </li>
        ))}
      </ol>
      <div className="flex justify-end gap-2">
        <Button variant="secondary" onClick={onDone}>
          Cancel
        </Button>
        <Button onClick={() => save.mutate()} loading={save.isPending} disabled={order === null}>
          Save order
        </Button>
      </div>
    </div>
  )
}
