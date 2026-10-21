import { api } from './client'

export type FieldKind =
  | 'text'
  | 'textarea'
  | 'html'
  | 'email'
  | 'emails'
  | 'url'
  | 'bool'
  | 'int'
  | 'color'
  | 'select'
  | 'time'
  | 'timezone'
  | 'file'
  | 'list'
  | 'phone'
  | 'coords'

export interface SettingField {
  key: string
  kind: FieldKind
  secret: boolean
  default: unknown
  options?: string[]
  min?: number
  max?: number
}

export interface SettingGroup {
  name: string
  fields: SettingField[]
}

export interface SecretView {
  configured: boolean
  masked: string
}

export type SettingValues = Record<string, unknown>

export interface ConnectionResult {
  ok: boolean
  message: string
}

export const SETTINGS_SCHEMA_KEY = ['settings', 'schema'] as const
export const settingsKey = (group: string) => ['settings', group] as const

export const settingsApi = {
  schema: (signal?: AbortSignal) => api.get<SettingGroup[]>('/settings', undefined, signal),
  get: (group: string, signal?: AbortSignal) => api.get<SettingValues>(`/settings/${group}`, undefined, signal),
  update: (group: string, values: SettingValues) => api.put<SettingValues>(`/settings/${group}`, values),
  test: (group: string) => api.post<ConnectionResult>(`/settings/${group}/test`),
}

export function isSecretView(value: unknown): value is SecretView {
  return typeof value === 'object' && value !== null && 'configured' in value && 'masked' in value
}
