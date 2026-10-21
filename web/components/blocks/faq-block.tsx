import { FaqAccordion } from '@/components/ui/faq-accordion'
import { list } from '@/lib/list'
import type { FaqSettings } from '@/lib/types'

export function FaqBlock({ settings }: { settings: FaqSettings }) {
  const items = list(settings.items).map((item) => ({ question: item.question, answerHtml: item.answer_html }))
  if (items.length === 0) {
    return null
  }
  return (
    <div className="mx-auto max-w-3xl">
      <FaqAccordion items={items} />
    </div>
  )
}
