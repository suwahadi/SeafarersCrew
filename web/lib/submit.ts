import type { ApplicationLimits, FieldErrors } from '@/lib/application-form'

export type SubmitFailure =
  | { kind: 'fields'; fields: FieldErrors; message: string; limits: Partial<ApplicationLimits> }
  | { kind: 'turnstile'; message: string }
  | { kind: 'banner'; code: string; message: string }
  | { kind: 'rate_limited'; message: string }
  | { kind: 'too_large'; message: string }
  | { kind: 'unavailable'; message: string }

export type SubmitResult<T> = { ok: true; data: T } | { ok: false; failure: SubmitFailure }

interface ErrorBody {
  error?: { code?: string; message?: string; fields?: Record<string, string> }
}

const BYTES_PER_MB = 1024 * 1024
const UNAVAILABLE = 'We could not send your request. Check your connection and try again.'
const GENERIC = 'Something went wrong while sending your request. Please try again in a moment.'

export function retryMessage(retryAfter: string | null): string {
  const seconds = Number.parseInt(retryAfter ?? '', 10)
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return 'Too many attempts. Please wait a few minutes and try again.'
  }
  const minutes = Math.max(1, Math.ceil(seconds / 60))
  return `Too many attempts. Please try again in about ${String(minutes)} ${minutes === 1 ? 'minute' : 'minutes'}.`
}

export function limitsFromFields(fields: FieldErrors): Partial<ApplicationLimits> {
  const limits: Partial<ApplicationLimits> = {}
  const size = /(\d+)\s*MB/i.exec(`${fields.cv ?? ''} ${fields.documents ?? ''}`)
  if (size?.[1]) {
    limits.maxFileBytes = Number(size[1]) * BYTES_PER_MB
  }
  const count = /at most (\d+)/i.exec(fields.documents ?? '')
  if (count?.[1]) {
    limits.maxDocuments = Number(count[1])
  }
  return limits
}

export function interpretFailure(
  status: number,
  body: ErrorBody | null,
  retryAfter: string | null,
  knownFields: readonly string[],
): SubmitFailure {
  const code = body?.error?.code ?? ''
  const message = body?.error?.message ?? ''

  if (status === 429) {
    return { kind: 'rate_limited', message: retryMessage(retryAfter) }
  }
  if (status === 413 || code === 'payload_too_large') {
    return {
      kind: 'too_large',
      message: 'Your files are too large to upload together. Remove or compress some files and try again.',
    }
  }
  if (code === 'turnstile_failed') {
    return { kind: 'turnstile', message: 'We could not verify that you are human. Please complete the check again.' }
  }
  if (status === 409 || status === 404) {
    return { kind: 'banner', code, message: message || GENERIC }
  }
  if (status === 422) {
    const raw = body?.error?.fields ?? {}
    const fields: FieldErrors = {}
    const unmatched: string[] = []
    for (const [name, text] of Object.entries(raw)) {
      if (knownFields.includes(name)) {
        fields[name] = text
      } else {
        unmatched.push(text)
      }
    }
    return {
      kind: 'fields',
      fields,
      message: unmatched.length > 0 ? unmatched.join(' ') : 'Please correct the highlighted fields and try again.',
      limits: limitsFromFields(fields),
    }
  }
  if (status === 503) {
    return { kind: 'unavailable', message: 'The service is temporarily unavailable. Please try again shortly.' }
  }
  return { kind: 'unavailable', message: GENERIC }
}

export async function submitRequest<T>(
  url: string,
  init: { body: FormData | string; headers?: Record<string, string> },
  knownFields: readonly string[],
): Promise<SubmitResult<T>> {
  let response: Response
  try {
    response = await fetch(url, { method: 'POST', credentials: 'omit', ...init })
  } catch {
    return { ok: false, failure: { kind: 'unavailable', message: UNAVAILABLE } }
  }
  let parsed: unknown = null
  try {
    parsed = await response.json()
  } catch {
    parsed = null
  }
  if (response.ok) {
    const data = (parsed as { data?: T } | null)?.data
    if (data !== undefined) {
      return { ok: true, data }
    }
    return { ok: false, failure: { kind: 'unavailable', message: GENERIC } }
  }
  return {
    ok: false,
    failure: interpretFailure(response.status, parsed as ErrorBody | null, response.headers.get('Retry-After'), knownFields),
  }
}
