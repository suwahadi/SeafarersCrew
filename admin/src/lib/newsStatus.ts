import type { NewsStatus } from '@/api/news'

export type NewsState = 'draft' | 'scheduled' | 'live'

export const NEWS_STATE_LABELS: Record<NewsState, string> = { draft: 'Draft', scheduled: 'Scheduled', live: 'Live' }

export function newsState(item: { status: NewsStatus; is_live: boolean }): NewsState {
  if (item.status === 'draft') return 'draft'
  return item.is_live ? 'live' : 'scheduled'
}

export function willBeScheduled(status: NewsStatus, publishAt: Date | null, now: Date = new Date()): boolean {
  return status === 'published' && publishAt !== null && publishAt.getTime() > now.getTime()
}
