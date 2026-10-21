import { z } from 'zod'
import { isSecretView, type SettingField, type SettingGroup, type SettingValues } from '@/api/settings'
import { titleCase } from '@/lib/format'

export type FieldValue = string | boolean | string[]
export type FormState = Record<string, FieldValue>

export const groupLabels: Record<string, string> = {
  general: 'General',
  company: 'Company',
  contact: 'Contact & Social',
  branding: 'Branding',
  seo: 'SEO',
  email: 'Email (Brevo)',
  storage: 'Object Storage',
  uploads: 'Uploads',
  security: 'Security',
  backup: 'Backup',
  alerts: 'Alerts',
  website: 'Website',
}

export const testableGroups = new Set(['email', 'storage', 'security'])

const fieldLabels: Record<string, string> = {
  brevo_api_key: 'Brevo API key',
  admin_notification_emails: 'Admin notification addresses',
  reply_to: 'Reply-to address',
  presign_minutes: 'Presigned URL lifetime (minutes)',
  max_file_size_mb: 'Max file size (MB)',
  max_applicant_files: 'Max files per applicant submission',
  webp_quality: 'WebP quality',
  max_image_dimension: 'Max image dimension (px)',
  pdf_preset: 'PDF compression preset',
  turnstile_site_key: 'Turnstile site key',
  turnstile_secret_key: 'Turnstile secret key',
  google_site_verification: 'Google Search Console verification',
  og_image_file_id: 'Default OG image',
  logo_light_file_id: 'Logo (light)',
  logo_dark_file_id: 'Logo (dark)',
  favicon_file_id: 'Favicon',
  x_url: 'X URL',
  use_path_style: 'Use path-style addressing',
  access_key_id: 'Access key ID',
  secret_access_key: 'Secret access key',
  run_time: 'Run time (WIB)',
  expiry_threshold_1: 'First expiry threshold (days)',
  expiry_threshold_2: 'Second expiry threshold (days)',
  expiry_threshold_3: 'Third expiry threshold (days)',
  hero_autoplay_seconds: 'Hero autoplay interval (seconds)',
  career_intro_html: 'Career page intro (HTML)',
  map_coordinates: 'Map coordinates (lat,lng)',
  meta_title_pattern: 'Meta title pattern',
  robots_indexing: 'Allow search engine indexing',
  maintenance_mode: 'Maintenance mode (public site)',
}

export function fieldLabel(key: string): string {
  return fieldLabels[key] ?? titleCase(key).replace(/\bUrl\b/g, 'URL').replace(/\bSeo\b/g, 'SEO').replace(/\bHtml\b/g, 'HTML')
}

function asStrings(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []
}

export function toFormState(group: SettingGroup, values: SettingValues): FormState {
  const out: FormState = {}
  for (const f of group.fields) {
    const v = values[f.key]
    if (f.secret || isSecretView(v)) {
      out[f.key] = ''
    } else if (f.kind === 'bool') {
      out[f.key] = v === true
    } else if (f.kind === 'list' || f.kind === 'emails') {
      out[f.key] = asStrings(v)
    } else if (typeof v === 'number') {
      out[f.key] = String(v)
    } else {
      out[f.key] = typeof v === 'string' ? v : ''
    }
  }
  return out
}

export function toPayload(group: SettingGroup, state: FormState): SettingValues {
  const out: SettingValues = {}
  for (const f of group.fields) {
    out[f.key] = fieldPayload(f, state[f.key])
  }
  return out
}

function fieldPayload(f: SettingField, v: FieldValue | undefined): unknown {
  if (f.kind === 'bool') return v === true
  if (f.kind === 'list' || f.kind === 'emails') return Array.isArray(v) ? v : []
  const s = typeof v === 'string' ? v.trim() : ''
  if (f.kind === 'int') return s === '' ? null : Number(s)
  return s
}

export function orderedListValue(f: SettingField, value: string[]): { item: string; enabled: boolean }[] {
  const options = f.options ?? []
  const enabled = value.filter((v) => options.length === 0 || options.includes(v))
  const disabled = options.filter((o) => !enabled.includes(o))
  return [...enabled.map((item) => ({ item, enabled: true })), ...disabled.map((item) => ({ item, enabled: false }))]
}

const EMAIL = z.email()

function scalarSchema(f: SettingField): z.ZodType<string> {
  const s = z.string().trim()
  const optional = (check: (v: string) => boolean, message: string) => s.refine((v) => v === '' || check(v), message)
  switch (f.kind) {
    case 'email':
      return optional((v) => EMAIL.safeParse(v).success, 'Enter a valid email address.')
    case 'url':
      return optional((v) => /^https?:\/\/[^\s]+\.[^\s]+$/.test(v), 'Enter a valid http or https URL.')
    case 'color':
      return optional((v) => /^#[0-9A-Fa-f]{6}$/.test(v), 'Use a hex color such as #0B2545.')
    case 'time':
      return s.refine((v) => /^([01]\d|2[0-3]):[0-5]\d$/.test(v), 'Use the HH:MM format.')
    case 'phone':
      return optional((v) => /^\+[1-9]\d{6,14}$/.test(v), 'Use the international format, for example +628123456789.')
    case 'coords':
      return optional((v) => /^-?\d{1,2}(\.\d+)?,\s*-?\d{1,3}(\.\d+)?$/.test(v), 'Use latitude,longitude, for example -6.2,106.8.')
    case 'int': {
      const message = f.min !== undefined && f.max !== undefined ? `Enter a whole number between ${f.min} and ${f.max}.` : 'Enter a whole number.'
      return s.refine((v) => {
        if (!/^-?\d+$/.test(v)) return false
        const n = Number(v)
        return (f.min === undefined || f.min === 0 || n >= f.min) && (f.max === undefined || f.max === 0 || n <= f.max)
      }, message)
    }
    default:
      return f.max !== undefined && f.max > 0 && f.kind !== 'html' ? s.max(f.max, `This field may not exceed ${f.max} characters.`) : s
  }
}

function fieldSchema(f: SettingField): z.ZodType<FieldValue> {
  if (f.kind === 'bool') return z.boolean()
  if (f.kind === 'emails') {
    return z.array(z.string()).refine((items) => items.every((i) => EMAIL.safeParse(i).success), 'Every address must be a valid email.')
  }
  if (f.kind === 'list') return z.array(z.string())
  return scalarSchema(f)
}

export function buildSettingsSchema(group: SettingGroup): z.ZodType<FormState, FormState> {
  return z.object(Object.fromEntries(group.fields.map((f) => [f.key, fieldSchema(f)]))) as unknown as z.ZodType<FormState, FormState>
}

export function isDirty(group: SettingGroup, state: FormState, initial: FormState): boolean {
  return group.fields.some((f) => JSON.stringify(state[f.key]) !== JSON.stringify(initial[f.key]))
}

export function secretStatus(values: SettingValues, key: string): { configured: boolean; masked: string } {
  const v = values[key]
  return isSecretView(v) ? { configured: v.configured, masked: v.masked } : { configured: false, masked: '' }
}
