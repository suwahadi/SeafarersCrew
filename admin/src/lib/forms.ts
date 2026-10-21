import type { FieldValues, Path, UseFormSetError } from 'react-hook-form'
import { ApiError } from '@/api/client'

export function applyServerErrors<T extends FieldValues>(err: unknown, setError: UseFormSetError<T>, fields: readonly Path<T>[]): string | null {
  if (!(err instanceof ApiError)) return err instanceof Error ? err.message : 'Something went wrong.'
  let unmatched = false
  for (const [key, message] of Object.entries(err.fields)) {
    const field = fields.find((f) => f === key)
    if (field) setError(field, { type: 'server', message })
    else unmatched = true
  }
  if (Object.keys(err.fields).length === 0 || unmatched) return err.message
  return null
}

export function emptyToNull(value: string): string | null {
  return value.trim() === '' ? null : value
}

export const PASSWORD_MIN = 12
