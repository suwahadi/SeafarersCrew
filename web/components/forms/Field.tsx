import type { ReactNode } from 'react'

export const CONTROL_CLASS =
  'block w-full rounded-md border border-slate-400 bg-white px-3 py-2.5 text-base text-ink transition-colors duration-150 ease-out placeholder:text-slate-500 focus-visible:border-accent-text aria-[invalid=true]:border-red-700'

export interface ControlProps {
  id: string
  'aria-invalid': true | undefined
  'aria-describedby': string | undefined
}

interface FieldProps {
  name: string
  idPrefix: string
  label?: string
  error?: string
  hint?: string
  extraDescribedBy?: string
  required?: boolean
  className?: string
  children: (control: ControlProps) => ReactNode
}

export function fieldId(prefix: string, name: string): string {
  return `${prefix}-${name}`
}

export function Field({
  name,
  idPrefix,
  label,
  error,
  hint,
  extraDescribedBy,
  required = false,
  className = '',
  children,
}: FieldProps) {
  const id = fieldId(idPrefix, name)
  const errorId = `${id}-error`
  const hintId = `${id}-hint`
  const describedBy = [error ? errorId : '', hint ? hintId : '', extraDescribedBy ?? ''].filter(Boolean).join(' ')
  return (
    <div className={className}>
      {label ? (
        <label htmlFor={id} className="mb-1.5 block text-sm font-semibold text-primary">
          {label}
          {required ? (
            <span aria-hidden="true" className="text-red-700">
              {' '}
              *
            </span>
          ) : null}
        </label>
      ) : null}
      {children({ id, 'aria-invalid': error ? true : undefined, 'aria-describedby': describedBy || undefined })}
      {hint ? (
        <p id={hintId} className="mt-1.5 text-sm text-muted">
          {hint}
        </p>
      ) : null}
      {error ? (
        <p id={errorId} role="alert" className="mt-1.5 text-sm font-medium text-red-700">
          {error}
        </p>
      ) : null}
    </div>
  )
}

export function Honeypot({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  return (
    <div className="hp-field" aria-hidden="true">
      <label>
        Leave this field empty
        <input
          type="text"
          name="website"
          tabIndex={-1}
          autoComplete="off"
          value={value}
          onChange={(event) => {
            onChange(event.target.value)
          }}
        />
      </label>
    </div>
  )
}

export function FormBanner({ message, id }: { message: string; id: string }) {
  return (
    <div id={id} role="alert" tabIndex={-1} className="rounded-md border border-red-300 bg-red-50 px-4 py-3 text-sm font-medium text-red-900">
      {message}
    </div>
  )
}

export function SubmitButton({ sending, label, sendingLabel }: { sending: boolean; label: string; sendingLabel: string }) {
  return (
    <button
      type="submit"
      disabled={sending}
      className="inline-flex items-center justify-center gap-2 rounded-md bg-primary px-6 py-3 text-base font-semibold text-on-primary transition-colors duration-200 ease-out hover:bg-primary-hover disabled:cursor-not-allowed disabled:opacity-70"
    >
      {sending ? <span className="spinner spinner-sm" aria-hidden="true" /> : null}
      {sending ? sendingLabel : label}
    </button>
  )
}

export function focusBanner(id: string): void {
  requestAnimationFrame(() => {
    const banner = document.getElementById(id)
    banner?.focus()
    banner?.scrollIntoView({ block: 'center' })
  })
}
