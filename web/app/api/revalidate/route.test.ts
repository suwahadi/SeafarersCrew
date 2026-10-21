import { NextRequest } from 'next/server'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const revalidateTag = vi.fn<(tag: string, profile: unknown) => void>()
vi.mock('next/cache', () => ({ revalidateTag: (tag: string, profile: unknown) => { revalidateTag(tag, profile) } }))

import { POST } from '@/app/api/revalidate/route'

const SECRET = 'a-very-long-revalidation-secret-0123456789'

function call(body: unknown, secret?: string): Promise<Response> {
  return POST(
    new NextRequest('http://localhost:3000/api/revalidate', {
      method: 'POST',
      headers: secret ? { 'x-revalidate-secret': secret, 'content-type': 'application/json' } : {},
      body: typeof body === 'string' ? body : JSON.stringify(body),
    }),
  )
}

describe('revalidate route', () => {
  beforeEach(() => {
    revalidateTag.mockClear()
    vi.stubEnv('REVALIDATE_SECRET', SECRET)
  })

  it('rejects a missing secret', async () => {
    expect((await call({ tags: ['site'] })).status).toBe(401)
    expect(revalidateTag).not.toHaveBeenCalled()
  })

  it('rejects a wrong secret', async () => {
    expect((await call({ tags: ['site'] }, 'wrong')).status).toBe(401)
    expect(revalidateTag).not.toHaveBeenCalled()
  })

  it('rejects every request when no secret is configured', async () => {
    vi.stubEnv('REVALIDATE_SECRET', '')
    expect((await call({ tags: ['site'] }, '')).status).toBe(401)
  })

  it('revalidates each known tag once and ignores unknown ones', async () => {
    const response = await call({ tags: ['site', 'news', 'news', 'bogus', 42] }, SECRET)
    expect(response.status).toBe(200)
    expect(revalidateTag.mock.calls.map((c) => c[0])).toEqual(['site', 'news'])
    expect((await response.json()) as unknown).toEqual({ revalidated: ['site', 'news'] })
  })

  it('returns 400 for invalid JSON and for bodies without valid tags', async () => {
    expect((await call('{nope', SECRET)).status).toBe(400)
    expect((await call({ tags: ['bogus'] }, SECRET)).status).toBe(400)
  })
})
