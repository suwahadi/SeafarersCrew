import type { Table } from '@tanstack/react-table'
import { useEffect, useRef, useState } from 'react'
import { IconButton } from '../ui/IconButton'

export function ColumnChooser<T>({ table }: { table: Table<T> }) {
  const [open, setOpen] = useState(false)
  const root = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!open) return
    const close = (e: PointerEvent | KeyboardEvent) => {
      if (e instanceof KeyboardEvent ? e.key === 'Escape' : !root.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('pointerdown', close)
    document.addEventListener('keydown', close)
    return () => {
      document.removeEventListener('pointerdown', close)
      document.removeEventListener('keydown', close)
    }
  }, [open])
  const columns = table.getAllLeafColumns().filter((c) => c.getCanHide())
  if (columns.length === 0) return null
  return (
    <div ref={root} className="relative">
      <IconButton icon="columns" label="Columns" bordered aria-expanded={open} aria-haspopup="true" onClick={() => setOpen((o) => !o)} />
      {open ? (
        <div className="absolute right-0 z-20 mt-1 max-h-80 w-56 max-w-[calc(100vw-2rem)] overflow-y-auto rounded-md bg-white p-2 shadow-lg ring-1 ring-slate-200">
          {columns.map((c) => (
            <label key={c.id} className="flex items-center gap-2 rounded px-2 py-1 text-sm hover:bg-slate-50">
              <input type="checkbox" checked={c.getIsVisible()} onChange={c.getToggleVisibilityHandler()} />
              {typeof c.columnDef.header === 'string' ? c.columnDef.header : c.id}
            </label>
          ))}
        </div>
      ) : null}
    </div>
  )
}
