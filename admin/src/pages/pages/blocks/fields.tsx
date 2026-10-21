import { useId, type InputHTMLAttributes, type ReactNode } from 'react'
import { Controller, get, useFormContext, type FieldPath } from 'react-hook-form'
import type { PageMedia } from '@/api/pages'
import { ImageField } from '@/components/cms/ImageField'
import { RichTextEditor } from '@/components/editor/RichTextEditor'
import { Button } from '@/components/ui/Button'
import { CheckboxField, SelectField, TextAreaField, TextField, inputClass, type Option } from '@/components/ui/Field'
import type { PageFormValues } from '@/lib/pagePayload'

export type FieldName = FieldPath<PageFormValues>

export interface BlockEditorProps {
  index: number
  media: PageMedia
}

function messageOf(value: unknown): string | undefined {
  if (value !== null && typeof value === 'object' && 'message' in value && typeof value.message === 'string') return value.message
  return undefined
}

export function useFieldError(name: FieldName): string | undefined {
  const { formState } = useFormContext<PageFormValues>()
  return messageOf(get(formState.errors, name)) ?? messageOf(get(formState.errors, `${name}.root`))
}

interface TextProps {
  name: FieldName
  label: string
  hint?: string
  required?: boolean
  placeholder?: string
  type?: InputHTMLAttributes<HTMLInputElement>['type']
  numeric?: boolean
  min?: number
  max?: number
}

export function BlockText({ name, label, hint, required, placeholder, type = 'text', numeric = false, min, max }: TextProps) {
  const { register } = useFormContext<PageFormValues>()
  const error = useFieldError(name)
  return (
    <TextField
      label={label}
      hint={hint}
      required={required}
      placeholder={placeholder}
      type={type}
      min={min}
      max={max}
      error={error}
      {...register(name, numeric ? { valueAsNumber: true } : undefined)}
    />
  )
}

export function BlockTextArea({ name, label, hint, rows = 3 }: { name: FieldName; label: string; hint?: string; rows?: number }) {
  const { register } = useFormContext<PageFormValues>()
  return <TextAreaField label={label} hint={hint} rows={rows} error={useFieldError(name)} {...register(name)} />
}

export function BlockSelect({ name, label, options, numeric = false }: { name: FieldName; label: string; options: Option[]; numeric?: boolean }) {
  const { register } = useFormContext<PageFormValues>()
  return <SelectField label={label} options={options} error={useFieldError(name)} {...register(name, numeric ? { valueAsNumber: true } : undefined)} />
}

export function BlockCheckbox({ name, label, hint }: { name: FieldName; label: string; hint?: string }) {
  const { register } = useFormContext<PageFormValues>()
  return <CheckboxField label={label} hint={hint} {...register(name)} />
}

export function BlockRich({ name, label, hint, minHeight = '8rem' }: { name: FieldName; label: string; hint?: string; minHeight?: string }) {
  const { control } = useFormContext<PageFormValues>()
  const error = useFieldError(name)
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => (
        <RichTextEditor
          label={label}
          hint={hint}
          value={typeof field.value === 'string' ? field.value : ''}
          onChange={field.onChange}
          withLayout={false}
          minHeight={minHeight}
          error={error}
        />
      )}
    />
  )
}

export function BlockImage({ name, label, media, required, hint }: { name: FieldName; label: string; media: PageMedia; required?: boolean; hint?: string }) {
  const { control } = useFormContext<PageFormValues>()
  const error = useFieldError(name)
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => {
        const value = typeof field.value === 'string' ? field.value : ''
        return <ImageField label={label} value={value} onChange={field.onChange} image={media[value]} required={required} hint={hint} error={error} />
      }}
    />
  )
}

const HEX = /^#[0-9a-fA-F]{6}$/

export function BlockColor({ name, label }: { name: FieldName; label: string }) {
  const { control } = useFormContext<PageFormValues>()
  const error = useFieldError(name)
  const id = useId()
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => {
        const value = typeof field.value === 'string' ? field.value : ''
        return (
          <div className="space-y-1">
            <label htmlFor={id} className="block text-sm font-medium text-slate-700">
              {label}
            </label>
            <div className="flex items-center gap-2">
              <input
                type="color"
                aria-label={`${label} picker`}
                value={HEX.test(value) ? value : '#ffffff'}
                onChange={(e) => field.onChange(e.target.value)}
                className="h-9 w-12 cursor-pointer rounded-md border-0 bg-white p-1 ring-1 ring-inset ring-slate-300"
              />
              <input
                id={id}
                type="text"
                value={value}
                maxLength={7}
                placeholder="#0A2540"
                aria-invalid={error ? true : undefined}
                aria-describedby={error ? `${id}-error` : undefined}
                onChange={(e) => field.onChange(e.target.value)}
                className={`w-32 ${inputClass}`}
              />
              {value === '' ? null : (
                <Button size="sm" variant="ghost" onClick={() => field.onChange('')}>
                  Clear
                </Button>
              )}
            </div>
            {error ? (
              <p id={`${id}-error`} role="alert" className="text-xs text-red-600">
                {error}
              </p>
            ) : null}
          </div>
        )
      }}
    />
  )
}

export const COLUMN_OPTIONS: Option[] = [
  { value: '2', label: '2 columns' },
  { value: '3', label: '3 columns' },
  { value: '4', label: '4 columns' },
]

interface ItemRowProps {
  label: string
  index: number
  count: number
  onMove: (to: number) => void
  onRemove: () => void
  children: ReactNode
}

export function ItemRow({ label, index, count, onMove, onRemove, children }: ItemRowProps) {
  return (
    <li className="rounded-md bg-slate-50 p-3 ring-1 ring-slate-200">
      <div className="mb-2 flex items-center justify-between gap-2">
        <span className="text-xs font-semibold uppercase tracking-wide text-slate-600">
          {label} {index + 1}
        </span>
        <div className="flex gap-1">
          <Button size="sm" variant="ghost" disabled={index === 0} aria-label={`Move ${label.toLowerCase()} ${index + 1} up`} onClick={() => onMove(index - 1)}>
            ↑
          </Button>
          <Button size="sm" variant="ghost" disabled={index === count - 1} aria-label={`Move ${label.toLowerCase()} ${index + 1} down`} onClick={() => onMove(index + 1)}>
            ↓
          </Button>
          <Button size="sm" variant="ghost" aria-label={`Remove ${label.toLowerCase()} ${index + 1}`} onClick={onRemove}>
            Remove
          </Button>
        </div>
      </div>
      <div className="space-y-3">{children}</div>
    </li>
  )
}

export function ListError({ message }: { message: string | undefined }) {
  return message ? (
    <p role="alert" className="text-xs text-red-600">
      {message}
    </p>
  ) : null
}
