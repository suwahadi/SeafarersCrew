'use client'

import { useEffect, useRef, useState, type KeyboardEvent, type MouseEvent } from 'react'
import { Icon } from '@/components/ui/icons'
import { SiteImage } from '@/components/ui/site-image'
import { nextIndex } from '@/lib/slider'
import type { GalleryItem, PublicImage } from '@/lib/types'

type Item = GalleryItem & { image: PublicImage }

interface LightboxGalleryProps {
  items: readonly GalleryItem[]
  columns?: number
  lightbox?: boolean
}

const COLUMN_CLASSES: Record<number, string> = {
  2: 'grid-cols-1 sm:grid-cols-2',
  3: 'grid-cols-2 md:grid-cols-3',
  4: 'grid-cols-2 md:grid-cols-3 lg:grid-cols-4',
}

function hasImage(item: GalleryItem): item is Item {
  return item.image !== null
}

export function LightboxGallery({ items, columns = 3, lightbox = true }: LightboxGalleryProps) {
  const visible = items.filter(hasImage)
  const [index, setIndex] = useState<number | null>(null)
  const dialogRef = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = dialogRef.current
    if (index !== null && dialog && !dialog.open) {
      dialog.showModal()
    }
  }, [index])

  const step = (direction: 1 | -1) => {
    setIndex((current) => (current === null ? null : nextIndex(current, visible.length, direction)))
  }

  const onKeyDown = (event: KeyboardEvent<HTMLDialogElement>) => {
    if (event.key === 'ArrowRight') {
      step(1)
    } else if (event.key === 'ArrowLeft') {
      step(-1)
    }
  }

  const onBackdropClick = (event: MouseEvent<HTMLDialogElement>) => {
    if (event.target === event.currentTarget) {
      dialogRef.current?.close()
    }
  }

  const current = index === null ? undefined : visible[index]
  const grid = COLUMN_CLASSES[columns] ?? COLUMN_CLASSES[3]

  return (
    <>
      <ul className={`grid gap-3 sm:gap-4 ${grid ?? ''}`}>
        {visible.map((item, position) => (
          <li key={item.id}>
            <figure className="overflow-hidden rounded-lg bg-surface">
              {lightbox ? (
                <button
                  type="button"
                  aria-label={`Open photo${item.caption ? `: ${item.caption}` : ''}`}
                  onClick={() => {
                    setIndex(position)
                  }}
                  className="group block w-full cursor-zoom-in overflow-hidden"
                >
                  <SiteImage
                    image={item.image}
                    sizes="(min-width: 1024px) 25vw, (min-width: 768px) 33vw, 50vw"
                    className="w-full object-cover transition-transform duration-300 ease-out group-hover:scale-105"
                    style={{ aspectRatio: '4 / 3' }}
                  />
                </button>
              ) : (
                <SiteImage
                  image={item.image}
                  sizes="(min-width: 1024px) 25vw, (min-width: 768px) 33vw, 50vw"
                  className="w-full object-cover"
                  style={{ aspectRatio: '4 / 3' }}
                />
              )}
              {item.caption ? <figcaption className="px-3 py-2 text-sm text-muted">{item.caption}</figcaption> : null}
            </figure>
          </li>
        ))}
      </ul>
      {lightbox ? (
        <dialog
          ref={dialogRef}
          className="lightbox"
          aria-label="Photo viewer"
          onClose={() => {
            setIndex(null)
          }}
          onKeyDown={onKeyDown}
          onClick={onBackdropClick}
        >
          {current ? (
            <div className="flex h-full flex-col items-center justify-center gap-3 p-4 sm:p-8">
              <SiteImage
                image={current.image}
                sizes="100vw"
                eager
                className="max-h-[80dvh] w-auto max-w-full rounded object-contain"
              />
              {current.caption ? <p className="max-w-2xl text-center text-sm text-white/90">{current.caption}</p> : null}
              <p className="text-xs text-white/60" aria-live="polite">
                {`${String((index ?? 0) + 1)} / ${String(visible.length)}`}
              </p>
            </div>
          ) : null}
          <button
            type="button"
            aria-label="Close"
            onClick={() => dialogRef.current?.close()}
            className="absolute right-3 top-3 inline-flex size-11 items-center justify-center rounded-full bg-white/15 hover:bg-white/25"
          >
            <Icon name="close" className="size-6" />
          </button>
          {visible.length > 1 ? (
            <>
              <button
                type="button"
                aria-label="Previous photo"
                onClick={() => {
                  step(-1)
                }}
                className="absolute left-3 top-1/2 inline-flex size-11 -translate-y-1/2 items-center justify-center rounded-full bg-white/15 hover:bg-white/25"
              >
                <Icon name="chevron-left" className="size-6" />
              </button>
              <button
                type="button"
                aria-label="Next photo"
                onClick={() => {
                  step(1)
                }}
                className="absolute right-3 top-1/2 inline-flex size-11 -translate-y-1/2 items-center justify-center rounded-full bg-white/15 hover:bg-white/25"
              >
                <Icon name="chevron-right" className="size-6" />
              </button>
            </>
          ) : null}
        </dialog>
      ) : null}
    </>
  )
}
