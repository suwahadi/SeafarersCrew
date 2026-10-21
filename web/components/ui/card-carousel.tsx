'use client'

import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { Icon } from '@/components/ui/icons'

interface CardCarouselProps {
  label: string
  children: ReactNode
}

const BUTTON =
  'inline-flex size-11 items-center justify-center rounded-full border border-line bg-white text-primary transition-colors duration-150 ease-out hover:bg-surface disabled:cursor-not-allowed disabled:opacity-40'

export function CardCarousel({ label, children }: CardCarouselProps) {
  const trackRef = useRef<HTMLDivElement>(null)
  const [canPrev, setCanPrev] = useState(false)
  const [canNext, setCanNext] = useState(true)

  const measure = useCallback(() => {
    const track = trackRef.current
    if (!track) {
      return
    }
    setCanPrev(track.scrollLeft > 4)
    setCanNext(track.scrollLeft + track.clientWidth < track.scrollWidth - 4)
  }, [])

  useEffect(() => {
    measure()
    window.addEventListener('resize', measure)
    return () => {
      window.removeEventListener('resize', measure)
    }
  }, [measure])

  const scrollByPage = (direction: 1 | -1) => {
    const track = trackRef.current
    if (track) {
      const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
      track.scrollBy({ left: direction * track.clientWidth * 0.9, behavior: reduced ? 'auto' : 'smooth' })
    }
  }

  return (
    <section aria-roledescription="carousel" aria-label={label}>
      <div
        ref={trackRef}
        onScroll={measure}
        className="scroll-track scroll-cards gap-6"
        tabIndex={0}
        role="group"
        aria-label={`${label} slides`}
      >
        {children}
      </div>
      <div className="mt-6 flex justify-center gap-3">
        <button type="button" aria-label="Previous" className={BUTTON} disabled={!canPrev} onClick={() => { scrollByPage(-1) }}>
          <Icon name="chevron-left" className="size-5" />
        </button>
        <button type="button" aria-label="Next" className={BUTTON} disabled={!canNext} onClick={() => { scrollByPage(1) }}>
          <Icon name="chevron-right" className="size-5" />
        </button>
      </div>
    </section>
  )
}
