import { Icon } from '@/components/ui/icons'
import { safeHref } from '@/lib/links'
import type { Site } from '@/lib/types'

export function WhatsAppButton({ whatsapp }: { whatsapp: Site['contact']['whatsapp'] }) {
  if (!whatsapp || whatsapp.number.trim() === '') {
    return null
  }
  const href = safeHref(whatsapp.url)
  if (href === '') {
    return null
  }
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      aria-label="Chat with us on WhatsApp"
      className="fixed bottom-5 right-5 z-30 inline-flex size-14 items-center justify-center rounded-full bg-[#25D366] text-white shadow-lg transition-transform duration-200 ease-out hover:scale-105 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#25D366]"
    >
      <Icon name="whatsapp" className="size-8" />
    </a>
  )
}
