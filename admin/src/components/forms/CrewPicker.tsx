import { useQuery } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { crewLookupApi } from '@/api/crewLookup'
import { useDebounced } from '@/hooks/useDebounced'
import { FieldShell, inputClass } from '../ui/Field'
import { Spinner } from '../ui/Spinner'

export interface PickedCrew {
  id: string
  name: string
}

interface CrewPickerProps {
  label: string
  value: PickedCrew | null
  onChange: (crew: PickedCrew | null) => void
  error?: string | undefined
  disabled?: boolean
  required?: boolean
}

export function CrewPicker({ label, value, onChange, error, disabled = false, required = false }: CrewPickerProps) {
  const id = useId()
  const [text, setText] = useState('')
  const [open, setOpen] = useState(false)
  const q = useDebounced(text.trim(), 300)
  const results = useQuery({
    queryKey: ['crew-lookup', q],
    queryFn: ({ signal }) => crewLookupApi.search(q, signal),
    enabled: open && q.length >= 2,
    staleTime: 30_000,
  })

  if (value) {
    return (
      <FieldShell id={id} label={label} error={error} required={required}>
        <div className="flex items-center justify-between gap-2 rounded-md bg-slate-50 px-3 py-1.5 text-sm ring-1 ring-slate-300">
          <span id={id}>
            {value.name} <span className="text-slate-500">({value.id})</span>
          </span>
          {disabled ? null : (
            <button type="button" className="text-xs text-navy-700 hover:underline" onClick={() => onChange(null)}>
              Change
            </button>
          )}
        </div>
      </FieldShell>
    )
  }

  const items = results.data ?? []
  return (
    <FieldShell id={id} label={label} error={error} required={required} hint="Type at least two characters of a name, Crew ID or passport number.">
      <div className="relative">
        <input
          id={id}
          role="combobox"
          aria-expanded={open && items.length > 0}
          aria-controls={`${id}-list`}
          aria-autocomplete="list"
          aria-invalid={error ? true : undefined}
          disabled={disabled}
          value={text}
          onChange={(e) => {
            setText(e.target.value)
            setOpen(true)
          }}
          onFocus={() => setOpen(true)}
          onBlur={() => window.setTimeout(() => setOpen(false), 150)}
          className={inputClass}
          placeholder="Search crew…"
        />
        {results.isFetching ? (
          <span className="absolute top-2 right-2 text-slate-400">
            <Spinner size="sm" label="Searching" />
          </span>
        ) : null}
        {open && items.length > 0 ? (
          <ul id={`${id}-list`} role="listbox" className="absolute z-30 mt-1 max-h-64 w-full overflow-y-auto rounded-md bg-white py-1 shadow-lg ring-1 ring-slate-200">
            {items.map((c) => (
              <li key={c.id} role="option" aria-selected={false}>
                <button
                  type="button"
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => {
                    onChange({ id: c.id, name: c.full_name })
                    setText('')
                    setOpen(false)
                  }}
                  className="flex w-full items-center justify-between gap-2 px-3 py-1.5 text-left text-sm hover:bg-slate-50"
                >
                  <span>
                    {c.full_name} <span className="text-slate-500">{c.id}</span>
                  </span>
                  <span className="text-xs text-slate-500">
                    {c.rank_name || '—'} · {c.status.name}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </FieldShell>
  )
}
