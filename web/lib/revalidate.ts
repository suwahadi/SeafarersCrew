import { createHash, timingSafeEqual } from 'node:crypto'
import { isCacheTag, type CacheTag } from '@/lib/tags'

function digest(value: string): Buffer {
  return createHash('sha256').update(value).digest()
}

export function secretsMatch(provided: string | null, expected: string): boolean {
  if (expected === '' || provided === null || provided === '') {
    return false
  }
  return timingSafeEqual(digest(provided), digest(expected))
}

export function parseTags(body: unknown): CacheTag[] {
  if (typeof body !== 'object' || body === null || !('tags' in body) || !Array.isArray(body.tags)) {
    return []
  }
  const unique = new Set<CacheTag>()
  for (const tag of body.tags as unknown[]) {
    if (isCacheTag(tag)) {
      unique.add(tag)
    }
  }
  return [...unique]
}
