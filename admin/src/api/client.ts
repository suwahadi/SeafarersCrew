export const API_BASE = '/api/admin/v1'

export interface ListMeta {
  page: number
  per_page: number
  total: number
}

export interface Envelope<T> {
  data: T
  meta?: ListMeta
}

export interface Paginated<T> {
  data: T
  meta: ListMeta
}

export interface ApiErrorBody {
  code: string
  message: string
  fields?: Record<string, string>
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly fields: Record<string, string>

  constructor(status: number, body: ApiErrorBody) {
    super(body.message)
    this.name = 'ApiError'
    this.status = status
    this.code = body.code
    this.fields = body.fields ?? {}
  }
}

export interface Job {
  id: string
  kind: string
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  result?: unknown
  error?: string
  attempts: number
  created_at: string
  finished_at?: string
}

type QueryValue = string | number | boolean | null | undefined | readonly (string | number)[]
export type Query = Record<string, QueryValue>

let csrfToken = ''
let onUnauthenticated: () => void = () => undefined

export function setCsrfToken(token: string): void {
  csrfToken = token
}

export function setUnauthenticatedHandler(handler: () => void): void {
  onUnauthenticated = handler
}

export function buildQuery(query?: Query): string {
  if (!query) return ''
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue
    if (Array.isArray(value)) {
      if (value.length > 0) params.set(key, value.join(','))
      continue
    }
    params.set(key, String(value))
  }
  const s = params.toString()
  return s ? `?${s}` : ''
}

interface RequestOptions {
  query?: Query
  body?: unknown
  form?: FormData
  signal?: AbortSignal
}

async function request<T>(method: string, path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  let body: BodyInit | undefined
  if (opts.form) {
    body = opts.form
  } else if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(opts.body)
  }
  if (method !== 'GET' && csrfToken) headers['X-CSRF-Token'] = csrfToken
  const res = await fetch(API_BASE + path + buildQuery(opts.query), {
    method,
    headers,
    body,
    credentials: 'include',
    signal: opts.signal,
  })
  if (res.status === 204) return { data: undefined } as T
  const text = await res.text()
  const parsed: unknown = text ? JSON.parse(text) : undefined
  if (!res.ok) {
    const err = (parsed as { error?: ApiErrorBody } | undefined)?.error ?? {
      code: 'http_error',
      message: `Request failed with status ${res.status}`,
    }
    if (res.status === 401) onUnauthenticated()
    throw new ApiError(res.status, err)
  }
  return parsed as T
}

export const api = {
  get: <T>(path: string, query?: Query, signal?: AbortSignal) => request<Envelope<T>>('GET', path, { query, signal }).then((r) => r.data),
  list: <T>(path: string, query?: Query, signal?: AbortSignal) => request<Paginated<T>>('GET', path, { query, signal }),
  post: <T>(path: string, body?: unknown, query?: Query) => request<Envelope<T>>('POST', path, { body, query }).then((r) => r.data),
  put: <T>(path: string, body?: unknown) => request<Envelope<T>>('PUT', path, { body }).then((r) => r.data),
  patch: <T>(path: string, body?: unknown) => request<Envelope<T>>('PATCH', path, { body }).then((r) => r.data),
  delete: (path: string) => request<Envelope<undefined>>('DELETE', path).then(() => undefined),
  upload: <T>(path: string, form: FormData) => request<Envelope<T>>('POST', path, { form }).then((r) => r.data),
  raw: async (path: string): Promise<Blob> => {
    const res = await fetch(API_BASE + path, { credentials: 'include' })
    if (!res.ok) throw new ApiError(res.status, { code: 'http_error', message: `Download failed (${res.status})` })
    return res.blob()
  },
}

export function fieldErrors(err: unknown): Record<string, string> {
  return err instanceof ApiError ? err.fields : {}
}

export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) return err.message
  if (err instanceof Error) return err.message
  return 'Something went wrong.'
}
