import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage } from '@/api/client'
import { fetchAllMasterRows, masterApi, OPTIONS_KEY, type MasterRecord } from '@/api/masters'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
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
  const [dragIndex, setDragIndex] = useState<number | null>(null)
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

  const move = (from: number, to: number) => setOrder(moveItem(items, from, to))

  if (rows.isPending) {
    return (
      <div className="flex justify-center p-6">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-600">Drag rows, or use the arrow buttons, to set the order used in every dropdown.</p>
      <ol className="divide-y divide-slate-100 rounded-md ring-1 ring-slate-200" aria-label="Order">
        {items.map((row, i) => (
          <li
            key={row.id}
            draggable
            onDragStart={() => setDragIndex(i)}
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault()
              if (dragIndex !== null) move(dragIndex, i)
              setDragIndex(null)
            }}
            onDragEnd={() => setDragIndex(null)}
            className={`flex items-center gap-3 px-3 py-2 text-sm ${dragIndex === i ? 'bg-navy-50' : 'bg-white'}`}
          >
            <span aria-hidden className="cursor-grab text-slate-400">⋮⋮</span>
            <span className="flex-1">{rowLabel(row)}</span>
            <button type="button" aria-label={`Move ${rowLabel(row)} up`} disabled={i === 0} onClick={() => move(i, i - 1)} className="rounded px-2 py-0.5 hover:bg-slate-100 disabled:opacity-30">
              ↑
            </button>
            <button
              type="button"
              aria-label={`Move ${rowLabel(row)} down`}
              disabled={i === items.length - 1}
              onClick={() => move(i, i + 1)}
              className="rounded px-2 py-0.5 hover:bg-slate-100 disabled:opacity-30"
            >
              ↓
            </button>
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
