import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import type { ReactElement } from 'react'
import { MemoryRouter } from 'react-router'
import { ToastProvider } from '@/components/ui/Toast'

export interface RecordedCall {
  method: string
  path: string
  body: unknown
}

export interface MockReply {
  status?: number
  body?: unknown
}

export type MockHandler = (call: RecordedCall) => MockReply | undefined

export function mockApi(handler: MockHandler): RecordedCall[] {
  const calls: RecordedCall[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn((input: string, init?: RequestInit) => {
      const path = input.replace('/api/admin/v1', '')
      const raw = typeof init?.body === 'string' ? init.body : undefined
      const call: RecordedCall = { method: init?.method ?? 'GET', path, body: raw ? (JSON.parse(raw) as unknown) : undefined }
      calls.push(call)
      const reply = handler(call) ?? { status: 404, body: { error: { code: 'not_found', message: 'No mock' } } }
      const status = reply.status ?? 200
      return Promise.resolve(new Response(JSON.stringify(reply.body ?? { data: null }), { status, headers: { 'Content-Type': 'application/json' } }))
    }),
  )
  return calls
}

export function renderApp(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  const view = render(
    <QueryClientProvider client={client}>
      <ToastProvider>
        <MemoryRouter>{ui}</MemoryRouter>
      </ToastProvider>
    </QueryClientProvider>,
  )
  return { ...view, client }
}
