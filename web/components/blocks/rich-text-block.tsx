import { RichHtml } from '@/components/ui/rich-html'
import type { RichTextSettings } from '@/lib/types'

export function RichTextBlock({ settings }: { settings: RichTextSettings }) {
  return <RichHtml html={settings.html} />
}
