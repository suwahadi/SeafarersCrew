import type { Table } from '@tanstack/react-table'
import { useState } from 'react'

export function ColumnChooser<T>({ table }: { table: Table<T> }) {
  const [open, setOpen] = useState(false)
  const columns = table.getAllLeafColumns().filter((c) => c.getCanHide())
  if (columns.length === 0) return null
  return (
    <div className="relative">
      <button
        type="button"
        aria-expanded={open}
        aria-haspopup="true"
        onClick={() => setOpen((o) => !o)}
        className="h-9 rounded-md bg-white px-3 text-sm ring-1 ring-inset ring-slate-300 hover:bg-slate-50"
      >
        Columns
      </button>
      {open ? (
        <div className="absolute right-0 z-20 mt-1 max-h-80 w-56 overflow-y-auto rounded-md bg-white p-2 shadow-lg ring-1 ring-slate-200">
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
