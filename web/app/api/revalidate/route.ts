import { revalidateTag } from 'next/cache'
import { NextResponse, type NextRequest } from 'next/server'
import { revalidateSecret } from '@/lib/env'
import { parseTags, secretsMatch } from '@/lib/revalidate'

export async function POST(request: NextRequest): Promise<NextResponse> {
  if (!secretsMatch(request.headers.get('x-revalidate-secret'), revalidateSecret())) {
    return NextResponse.json({ error: 'unauthorized' }, { status: 401 })
  }

  let body: unknown
  try {
    body = await request.json()
  } catch {
    return NextResponse.json({ error: 'invalid_json' }, { status: 400 })
  }

  const tags = parseTags(body)
  if (tags.length === 0) {
    return NextResponse.json({ error: 'no_valid_tags' }, { status: 400 })
  }

  for (const tag of tags) {
    revalidateTag(tag, { expire: 0 })
  }
  return NextResponse.json({ revalidated: tags })
}
