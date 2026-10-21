import { ButtonLink } from '@/components/ui/button'
import { RichHtml } from '@/components/ui/rich-html'
import { SiteImage } from '@/components/ui/site-image'
import type { ImageTextSettings } from '@/lib/types'

export function ImageTextBlock({ settings }: { settings: ImageTextSettings }) {
  const imageRight = settings.image_position === 'right'
  return (
    <div className="grid items-center gap-8 md:grid-cols-2 md:gap-12">
      {settings.image ? (
        <div className={imageRight ? 'md:order-2' : ''}>
          <SiteImage
            image={settings.image}
            sizes="(min-width: 1152px) 560px, (min-width: 768px) 45vw, 100vw"
            className="w-full rounded-lg object-cover"
          />
        </div>
      ) : null}
      <div className={settings.image ? '' : 'md:col-span-2'}>
        {settings.heading ? (
          <h2 className="text-2xl font-bold tracking-tight text-primary sm:text-3xl">{settings.heading}</h2>
        ) : null}
        <RichHtml html={settings.text_html} className="mt-4" />
        {settings.button_label && settings.button_link ? (
          <ButtonLink href={settings.button_link} className="mt-6">
            {settings.button_label}
          </ButtonLink>
        ) : null}
      </div>
    </div>
  )
}
