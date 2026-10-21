import { forwardRef, useId, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from 'react'

const control =
  'block w-full rounded-md border-0 bg-white px-3 py-1.5 text-sm text-slate-900 ring-1 ring-inset ring-slate-300 placeholder:text-slate-400 focus:ring-2 focus:ring-inset focus:ring-navy-500 disabled:bg-slate-100 disabled:text-slate-500 aria-[invalid=true]:ring-red-500'

interface FieldShellProps {
  id: string
  label: string
  error?: string | undefined
  hint?: string | undefined
  required?: boolean | undefined
  children: ReactNode
}

export function FieldShell({ id, label, error, hint, required, children }: FieldShellProps) {
  return (
    <div className="space-y-1">
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
        {required ? <span className="text-red-600"> *</span> : null}
      </label>
      {children}
      {error ? (
        <p id={`${id}-error`} role="alert" className="text-xs text-red-600">
          {error}
        </p>
      ) : hint ? (
        <p id={`${id}-hint`} className="text-xs text-slate-500">
          {hint}
        </p>
      ) : null}
    </div>
  )
}

interface CommonProps {
  label: string
  error?: string | undefined
  hint?: string | undefined
}

export const TextField = forwardRef<HTMLInputElement, CommonProps & InputHTMLAttributes<HTMLInputElement>>(function TextField(
  { label, error, hint, id, required, className = '', ...rest },
  ref,
) {
  const auto = useId()
  const fid = id ?? auto
  return (
    <FieldShell id={fid} label={label} error={error} hint={hint} required={required}>
      <input
        ref={ref}
        id={fid}
        required={required}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${fid}-error` : hint ? `${fid}-hint` : undefined}
        className={`${control} ${className}`}
        {...rest}
      />
    </FieldShell>
  )
})

export const TextAreaField = forwardRef<HTMLTextAreaElement, CommonProps & TextareaHTMLAttributes<HTMLTextAreaElement>>(function TextAreaField(
  { label, error, hint, id, required, className = '', rows = 3, ...rest },
  ref,
) {
  const auto = useId()
  const fid = id ?? auto
  return (
    <FieldShell id={fid} label={label} error={error} hint={hint} required={required}>
      <textarea
        ref={ref}
        id={fid}
        rows={rows}
        required={required}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${fid}-error` : undefined}
        className={`${control} ${className}`}
        {...rest}
      />
    </FieldShell>
  )
})

export interface Option {
  value: string
  label: string
}

export const SelectField = forwardRef<HTMLSelectElement, CommonProps & SelectHTMLAttributes<HTMLSelectElement> & { options: Option[]; placeholder?: string }>(
  function SelectField({ label, error, hint, id, required, options, placeholder, className = '', ...rest }, ref) {
    const auto = useId()
    const fid = id ?? auto
    return (
      <FieldShell id={fid} label={label} error={error} hint={hint} required={required}>
        <select
          ref={ref}
          id={fid}
          required={required}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${fid}-error` : undefined}
          className={`${control} ${className}`}
          {...rest}
        >
          {placeholder !== undefined ? <option value="">{placeholder}</option> : null}
          {options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </FieldShell>
    )
  },
)

export const CheckboxField = forwardRef<HTMLInputElement, { label: string; hint?: string } & InputHTMLAttributes<HTMLInputElement>>(function CheckboxField(
  { label, hint, id, ...rest },
  ref,
) {
  const auto = useId()
  const fid = id ?? auto
  return (
    <div className="flex items-start gap-2">
      <input ref={ref} id={fid} type="checkbox" className="mt-0.5 h-4 w-4 rounded border-slate-300 text-navy-800 focus:ring-navy-500" {...rest} />
      <div>
        <label htmlFor={fid} className="text-sm font-medium text-slate-700">
          {label}
        </label>
        {hint ? <p className="text-xs text-slate-500">{hint}</p> : null}
      </div>
    </div>
  )
})

export const inputClass = control
