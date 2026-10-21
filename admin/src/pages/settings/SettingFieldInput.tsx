import { useMemo } from 'react'
import type { SettingField } from '@/api/settings'
import { FileUploadField } from '@/components/forms/FileUploadField'
import { TagInput } from '@/components/forms/TagInput'
import { RichTextEditor } from '@/components/editor/RichTextEditor'
import { CheckboxField, FieldShell, inputClass, SelectField, TextAreaField, TextField } from '@/components/ui/Field'
import { titleCase } from '@/lib/format'
import { fieldLabel, orderedListValue, type FieldValue } from './settingsForm'

const FALLBACK_TIMEZONES = ['Asia/Jakarta', 'Asia/Singapore', 'Asia/Manila', 'Asia/Dubai', 'Europe/London', 'UTC']

export function supportedTimezones(current: string): string[] {
  let zones: string[] = FALLBACK_TIMEZONES
  if (typeof Intl.supportedValuesOf === 'function') {
    try {
      zones = Intl.supportedValuesOf('timeZone')
    } catch {
      zones = FALLBACK_TIMEZONES
    }
  }
  const all = zones.includes('UTC') ? zones : [...zones, 'UTC']
  return current !== '' && !all.includes(current) ? [current, ...all] : all
}

const optionLabels: Record<string, string> = { aws_s3: 'AWS S3', s3_compatible: 'S3-compatible storage' }

function optionLabel(option: string): string {
  if (optionLabels[option]) return optionLabels[option]
  return /[A-Z0-9/-]/.test(option) ? option : titleCase(option)
}

interface SettingFieldInputProps {
  field: SettingField
  value: FieldValue
  onChange: (value: FieldValue) => void
  error?: string | undefined
  secret?: { configured: boolean; masked: string }
}

export function SettingFieldInput({ field, value, onChange, error, secret }: SettingFieldInputProps) {
  const label = fieldLabel(field.key)
  const text = typeof value === 'string' ? value : ''
  const list = Array.isArray(value) ? value : []
  const zones = useMemo(() => (field.kind === 'timezone' ? supportedTimezones(text) : []), [field.kind, text])

  if (field.secret) {
    return (
      <SecretInput
        label={label}
        value={text}
        onChange={onChange}
        error={error}
        configured={secret?.configured ?? false}
        masked={secret?.masked ?? ''}
      />
    )
  }

  switch (field.kind) {
    case 'bool':
      return <CheckboxField label={label} checked={value === true} onChange={(e) => onChange(e.target.checked)} />
    case 'textarea':
      return <TextAreaField label={label} value={text} error={error} onChange={(e) => onChange(e.target.value)} />
    case 'html':
      return <RichTextEditor label={label} value={text} onChange={onChange} error={error} withLayout={false} minHeight="8rem" />
    case 'emails':
      return (
        <TagInput
          label={label}
          value={list}
          onChange={onChange}
          error={error}
          placeholder="name@example.com, press Enter"
          validate={(item) => (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(item) ? null : 'Enter a valid email address.')}
        />
      )
    case 'list':
      return field.options && field.options.length > 0 ? (
        <OrderedOptionList label={label} field={field} value={list} onChange={onChange} />
      ) : (
        <TagInput label={label} value={list} onChange={onChange} error={error} />
      )
    case 'select':
      return <SelectField label={label} value={text} options={(field.options ?? []).map((o) => ({ value: o, label: optionLabel(o) }))} error={error} onChange={(e) => onChange(e.target.value)} />
    case 'timezone':
      return <SelectField label={label} value={text} options={zones.map((z) => ({ value: z, label: z }))} error={error} onChange={(e) => onChange(e.target.value)} />
    case 'file':
      return <FileUploadField label={label} context="logo" value={text} onChange={onChange} error={error} />
    case 'color':
      return (
        <div className="flex items-end gap-2">
          <input
            type="color"
            aria-label={`${label} picker`}
            value={/^#[0-9A-Fa-f]{6}$/.test(text) ? text : '#0B2545'}
            onChange={(e) => onChange(e.target.value.toUpperCase())}
            className="h-9 w-12 rounded border border-slate-300"
          />
          <div className="flex-1">
            <TextField label={label} value={text} error={error} onChange={(e) => onChange(e.target.value)} />
          </div>
        </div>
      )
    case 'int':
      return (
        <TextField
          label={label}
          type="number"
          inputMode="numeric"
          min={field.min}
          max={field.max}
          value={text}
          error={error}
          hint={field.min !== undefined && field.max !== undefined ? `Between ${field.min} and ${field.max}` : undefined}
          onChange={(e) => onChange(e.target.value)}
        />
      )
    case 'time':
      return <TextField label={label} type="time" value={text} error={error} onChange={(e) => onChange(e.target.value)} />
    case 'email':
      return <TextField label={label} type="email" value={text} error={error} onChange={(e) => onChange(e.target.value)} />
    case 'url':
      return <TextField label={label} type="url" value={text} error={error} placeholder="https://" onChange={(e) => onChange(e.target.value)} />
    case 'phone':
      return <TextField label={label} type="tel" value={text} error={error} placeholder="+628123456789" onChange={(e) => onChange(e.target.value)} />
    case 'coords':
      return <TextField label={label} value={text} error={error} placeholder="-6.200000,106.816666" onChange={(e) => onChange(e.target.value)} />
    default:
      return <TextField label={label} value={text} error={error} maxLength={field.max && field.max > 0 ? field.max : undefined} onChange={(e) => onChange(e.target.value)} />
  }
}

function SecretInput({ label, value, onChange, error, configured, masked }: { label: string; value: string; onChange: (v: string) => void; error?: string | undefined; configured: boolean; masked: string }) {
  return (
    <div className="space-y-1">
      <p className="text-sm font-medium text-slate-700">{label}</p>
      <p className="text-xs text-slate-600" data-testid={`secret-status-${label}`}>
        {configured ? (
          <>
            Configured: <span className="font-mono">{masked}</span>
          </>
        ) : (
          'Not configured'
        )}
      </p>
      <FieldShell id={`secret-${label}`} label={configured ? `Replace ${label}` : `Set ${label}`} error={error} hint={configured ? 'Leave empty to keep the current value.' : undefined}>
        <input
          id={`secret-${label}`}
          type="password"
          autoComplete="new-password"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          aria-invalid={error ? true : undefined}
          className={inputClass}
        />
      </FieldShell>
    </div>
  )
}

function OrderedOptionList({ label, field, value, onChange }: { label: string; field: SettingField; value: string[]; onChange: (v: string[]) => void }) {
  const items = orderedListValue(field, value)
  const commit = (next: { item: string; enabled: boolean }[]) => onChange(next.filter((i) => i.enabled).map((i) => i.item))
  const move = (from: number, to: number) => {
    const next = [...items]
    const [moved] = next.splice(from, 1)
    if (!moved) return
    next.splice(to, 0, moved)
    commit(next)
  }
  const toggle = (index: number) => commit(items.map((it, i) => (i === index ? { ...it, enabled: !it.enabled } : it)))

  return (
    <fieldset className="space-y-1">
      <legend className="text-sm font-medium text-slate-700">{label}</legend>
      <ol className="divide-y divide-slate-100 rounded-md ring-1 ring-slate-200">
        {items.map((it, i) => {
          const name = titleCase(it.item)
          const canUp = it.enabled && i > 0 && items[i - 1]?.enabled === true
          const canDown = it.enabled && items[i + 1]?.enabled === true
          return (
            <li key={it.item} className="flex items-center gap-3 px-3 py-2 text-sm">
              <input type="checkbox" aria-label={`Show ${name}`} checked={it.enabled} onChange={() => toggle(i)} className="h-4 w-4 rounded border-slate-300 text-navy-800" />
              <span className={`flex-1 ${it.enabled ? '' : 'text-slate-400'}`}>{name}</span>
              <button type="button" aria-label={`Move ${name} up`} disabled={!canUp} onClick={() => move(i, i - 1)} className="rounded px-2 py-0.5 hover:bg-slate-100 disabled:opacity-30">
                ↑
              </button>
              <button type="button" aria-label={`Move ${name} down`} disabled={!canDown} onClick={() => move(i, i + 1)} className="rounded px-2 py-0.5 hover:bg-slate-100 disabled:opacity-30">
                ↓
              </button>
            </li>
          )
        })}
      </ol>
    </fieldset>
  )
}
