import { useId, useState, type KeyboardEvent } from 'react'
import { inputClass } from '../ui/Field'

interface TagInputProps {
  label: string
  value: string[]
  onChange: (value: string[]) => void
  placeholder?: string
  validate?: (item: string) => string | null
  error?: string | undefined
  hint?: string | undefined
}

export function TagInput({ label, value, onChange, placeholder = 'Type and press Enter', validate, error, hint }: TagInputProps) {
  const id = useId()
  const [draft, setDraft] = useState('')
  const [draftError, setDraftError] = useState<string | null>(null)

  const add = () => {
    const item = draft.trim()
    if (item === '') return
    const problem = validate?.(item) ?? null
    if (problem) {
      setDraftError(problem)
      return
    }
    if (!value.includes(item)) onChange([...value, item])
    setDraft('')
    setDraftError(null)
  }

  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      add()
    } else if (e.key === 'Backspace' && draft === '' && value.length > 0) {
      onChange(value.slice(0, -1))
    }
  }

  const shown = draftError ?? error
  return (
    <div className="space-y-1">
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <ul className="flex flex-wrap gap-1" aria-label={`${label} values`}>
        {value.map((item) => (
          <li key={item} className="inline-flex items-center gap-1 rounded-full bg-navy-50 px-2 py-0.5 text-xs text-navy-800">
            {item}
            <button type="button" aria-label={`Remove ${item}`} onClick={() => onChange(value.filter((v) => v !== item))} className="text-navy-500 hover:text-red-600">
              ×
            </button>
          </li>
        ))}
      </ul>
      <input
        id={id}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={onKey}
        onBlur={add}
        placeholder={placeholder}
        aria-invalid={shown ? true : undefined}
        className={inputClass}
      />
      {shown ? (
        <p role="alert" className="text-xs text-red-600">
          {shown}
        </p>
      ) : hint ? (
        <p className="text-xs text-slate-500">{hint}</p>
      ) : null}
    </div>
  )
}
