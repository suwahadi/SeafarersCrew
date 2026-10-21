import 'server-only'
import { notFound, permanentRedirect } from 'next/navigation'
import { findRedirect } from '@/lib/api'
import { safeHref } from '@/lib/links'
import { normalizePath } from '@/lib/seo'

export async function notFoundOrRedirect(path: string): Promise<never> {
  const rule = await findRedirect(path)
  const target = rule ? safeHref(rule.to_path) : ''
  if (target !== '' && normalizePath(target) !== normalizePath(path)) {
    permanentRedirect(target)
  }
  notFound()
}
