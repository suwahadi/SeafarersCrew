import { useEffect, useId, useRef, useState } from 'react'
import type { Option } from '../ui/Field'

interface MultiSelectProps {
  label: string
  options: Option[]
  value: string[]
  onChange: (value: string[]) => void
  placeholder?: string
}

export function MultiSelect({ label, options, value, onChange, placeholder = 'Any' }: MultiSelectProps) {
  const [open, setOpen] = useState(false)
  const id = useId()
  const root = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const close = (e: MouseEvent) => {
      if (root.current && !root.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [open])

  const selected = options.filter((o) => value.includes(o.value))
  const summary = selected.length === 0 ? placeholder : selected.length <= 2 ? selected.map((o) => o.label).join(', ') : `${selected.length} selected`

  const toggle = (v: string) => onChange(value.includes(v) ? value.filter((x) => x !== v) : [...value, v])

  return (
    <div ref={root} className="relative space-y-1">
      <span id={`${id}-label`} className="block text-sm font-medium text-slate-700">
        {label}
      </span>
      <button
        type="button"
        aria-labelledby={`${id}-label`}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') setOpen(false)
        }}
        className="flex h-[34px] w-full items-center justify-between rounded-md bg-white px-3 text-left text-sm ring-1 ring-inset ring-slate-300 focus:ring-2 focus:ring-navy-500"
      >
        <span className={selected.length === 0 ? 'text-slate-400' : 'truncate text-slate-900'}>{summary}</span>
        <span aria-hidden className="text-slate-400">
          ▾
        </span>
      </button>
      {open ? (
        <div
          role="listbox"
          aria-multiselectable="true"
          aria-labelledby={`${id}-label`}
          onKeyDown={(e) => {
            if (e.key === 'Escape') setOpen(false)
          }}
          className="absolute z-30 mt-1 max-h-64 w-full overflow-y-auto rounded-md bg-white p-1 shadow-lg ring-1 ring-slate-200"
        >
          {options.length === 0 ? <p className="px-2 py-1 text-sm text-slate-500">No options</p> : null}
          {options.map((o) => (
            <label key={o.value} className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 text-sm hover:bg-slate-50">
              <input type="checkbox" checked={value.includes(o.value)} onChange={() => toggle(o.value)} />
              {o.label}
            </label>
          ))}
          {value.length > 0 ? (
            <button type="button" onClick={() => onChange([])} className="mt-1 w-full rounded px-2 py-1 text-left text-xs text-navy-700 hover:bg-slate-50">
              Clear selection
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
