import { Icon, hasIcon } from '@/components/ui/icons'
import { SiteImage } from '@/components/ui/site-image'
import { SmartLink } from '@/components/ui/smart-link'
import { safeHref } from '@/lib/links'
import { list } from '@/lib/list'
import type { CardGridSettings } from '@/lib/types'

const COLUMNS: Record<number, string> = {
  2: 'sm:grid-cols-2',
  3: 'sm:grid-cols-2 lg:grid-cols-3',
  4: 'sm:grid-cols-2 lg:grid-cols-4',
}

interface CardGridProps {
  settings: CardGridSettings
  titleLevel?: 'h2' | 'h3'
}

export function CardGridBlock({ settings, titleLevel: Heading = 'h2' }: CardGridProps) {
  const cards = list(settings.cards)
  if (cards.length === 0) {
    return null
  }
  return (
    <ul className={`grid gap-6 ${COLUMNS[settings.columns] ?? COLUMNS[3] ?? ''}`}>
      {cards.map((card, index) => {
        const linked = safeHref(card.link) !== ''
        return (
          <li
            key={`${card.title}-${String(index)}`}
            className="relative flex flex-col overflow-hidden rounded-lg border border-line bg-white transition-shadow duration-200 ease-out hover:shadow-md"
          >
            {card.image ? (
              <div className="aspect-[16/10] overflow-hidden bg-surface">
                <SiteImage
                  image={card.image}
                  sizes="(min-width: 1024px) 384px, (min-width: 640px) 50vw, 100vw"
                  alt=""
                  className="size-full object-cover"
                />
              </div>
            ) : null}
            <div className="flex flex-1 flex-col p-6">
              {!card.image && hasIcon(card.icon) ? (
                <span className="mb-4 inline-flex size-12 items-center justify-center rounded-lg bg-accent/15 text-accent-text">
                  <Icon name={card.icon} className="size-6" />
                </span>
              ) : null}
              <Heading className="text-lg font-bold text-primary">
                {linked ? (
                  <SmartLink href={card.link} className="after:absolute after:inset-0 hover:text-accent-text">
                    {card.title}
                  </SmartLink>
                ) : (
                  card.title
                )}
              </Heading>
              {card.text ? <p className="mt-2 text-sm leading-6 text-muted">{card.text}</p> : null}
            </div>
          </li>
        )
      })}
    </ul>
  )
}
