import { z } from 'zod'
import type { MasterRecord } from '@/api/masters'
import { isValidIMO } from '@/lib/imo'
import type { FieldDef } from './config'

export type FormValues = Record<string, string | boolean>

function fieldSchema(f: FieldDef): z.ZodType<string | boolean> {
  if (f.type === 'checkbox') return z.boolean()
  let s = z.string().trim()
  if (f.maxLength !== undefined) s = s.max(f.maxLength, `This field may not exceed ${f.maxLength} characters.`)
  const base = f.required ? s.min(1, 'This field is required.') : s
  switch (f.type) {
    case 'email':
      return base.refine((v) => v === '' || z.email().safeParse(v).success, 'Enter a valid email address.')
    case 'url':
      return base.refine((v) => v === '' || /^https?:\/\/[^\s]+\.[^\s]+$/.test(v), 'Enter a valid http or https URL.')
    case 'imo':
      return base.refine((v) => v === '' || isValidIMO(v), 'The IMO number must be 7 digits with a valid check digit.')
    case 'color':
      return base.refine((v) => v === '' || /^#[0-9A-Fa-f]{6}$/.test(v), 'Use a hex color such as #2563EB.')
    case 'number':
      return base.refine((v) => {
        if (v === '') return true
        if (!/^-?\d+$/.test(v)) return false
        const n = Number(v)
        return (f.min === undefined || n >= f.min) && (f.max === undefined || n <= f.max)
      }, rangeMessage(f))
    case 'decimal':
      return base.refine((v) => v === '' || /^\d{1,10}(\.\d{1,2})?$/.test(v), 'Enter a number with up to 2 decimals.')
    default:
      return base
  }
}

function rangeMessage(f: FieldDef): string {
  if (f.min !== undefined && f.max !== undefined) return `Enter a whole number between ${f.min} and ${f.max}.`
  return 'Enter a whole number.'
}

export function buildSchema(fields: FieldDef[]): z.ZodType<FormValues, FormValues> {
  return z.object(Object.fromEntries(fields.map((f) => [f.name, fieldSchema(f)]))) as unknown as z.ZodType<FormValues, FormValues>
}

export function defaultValues(fields: FieldDef[], row?: MasterRecord): FormValues {
  const out: FormValues = {}
  for (const f of fields) {
    const v = row?.[f.name]
    if (f.type === 'checkbox') {
      out[f.name] = typeof v === 'boolean' ? v : f.defaultValue === true
      continue
    }
    if (v === null || v === undefined) out[f.name] = row ? '' : typeof f.defaultValue === 'string' ? f.defaultValue : ''
    else out[f.name] = typeof v === 'string' || typeof v === 'number' ? String(v) : ''
  }
  return out
}

export function toPayload(fields: FieldDef[], values: FormValues, row?: MasterRecord): Record<string, unknown> {
  const out: Record<string, unknown> = { is_active: row ? row.is_active : true }
  for (const f of fields) {
    const v = values[f.name]
    if (f.type === 'checkbox') {
      out[f.name] = v === true
      continue
    }
    const s = typeof v === 'string' ? v.trim() : ''
    switch (f.type) {
      case 'reference':
      case 'logo':
        out[f.name] = s === '' ? null : s
        break
      case 'number':
        out[f.name] = s === '' ? null : Number(s)
        break
      case 'decimal':
        out[f.name] = s === '' ? null : s
        break
      default:
        out[f.name] = f.upper ? s.toUpperCase() : s
    }
  }
  return out
}
