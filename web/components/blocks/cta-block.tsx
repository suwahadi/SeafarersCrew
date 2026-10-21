import { ButtonLink } from '@/components/ui/button'
import { SiteImage } from '@/components/ui/site-image'
import { parseHex, readableTextOn } from '@/lib/contrast'
import type { CtaSettings } from '@/lib/types'

export function CtaBlock({ settings }: { settings: CtaSettings }) {
  const hasImage = settings.background_image !== null
  const customColor = !hasImage && parseHex(settings.background_color) ? settings.background_color : ''
  return (
    <div
      className="relative isolate overflow-hidden rounded-xl bg-primary px-6 py-12 text-center text-on-primary sm:px-12 sm:py-16"
      style={customColor ? { backgroundColor: customColor, color: readableTextOn(customColor) } : undefined}
    >
      {settings.background_image ? (
        <>
          <SiteImage
            image={settings.background_image}
            sizes="(min-width: 1152px) 1100px, 100vw"
            alt=""
            className="absolute inset-0 -z-20 size-full object-cover"
          />
          <div aria-hidden="true" className="absolute inset-0 -z-10 bg-primary/80" />
        </>
      ) : null}
      {settings.heading ? <h2 className="text-2xl font-bold tracking-tight sm:text-3xl">{settings.heading}</h2> : null}
      {settings.text ? <p className="mx-auto mt-3 max-w-2xl text-lg opacity-90">{settings.text}</p> : null}
      {settings.button_label && settings.button_link ? (
        <ButtonLink href={settings.button_link} variant="accent" className="mt-8">
          {settings.button_label}
        </ButtonLink>
      ) : null}
    </div>
  )
}
