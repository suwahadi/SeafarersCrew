import type { NewsStatus } from '@/api/news'
import { Badge, type Tone } from '@/components/ui/Layout'
import { NEWS_STATE_LABELS, newsState, type NewsState } from '@/lib/newsStatus'

const tones: Record<NewsState, Tone> = { draft: 'neutral', scheduled: 'yellow', live: 'green' }

export function NewsStatusBadge({ news }: { news: { status: NewsStatus; is_live: boolean } }) {
  const state = newsState(news)
  return <Badge tone={tones[state]}>{NEWS_STATE_LABELS[state]}</Badge>
}
