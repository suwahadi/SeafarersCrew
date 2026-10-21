'use client'

import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { Icon } from '@/components/ui/icons'
import { usePageHidden, usePrefersReducedMotion } from '@/components/ui/use-browser-state'
import { autoplayDelayMs, indexFromScroll, nextIndex, shouldAutoplay } from '@/lib/slider'

interface HeroSliderProps {
  count: number
  autoplaySeconds: number
  children: ReactNode
}

const CONTROL =
  'inline-flex size-11 items-center justify-center rounded-full bg-primary/70 text-white transition-colors duration-150 ease-out hover:bg-primary focus-visible:outline-white'

export function HeroSlider({ count, autoplaySeconds, children }: HeroSliderProps) {
  const trackRef = useRef<HTMLDivElement>(null)
  const [index, setIndex] = useState(0)
  const [hovered, setHovered] = useState(false)
  const [focused, setFocused] = useState(false)
  const [userPaused, setUserPaused] = useState(false)
  const reducedMotion = usePrefersReducedMotion()
  const hidden = usePageHidden()

  const goTo = useCallback(
    (target: number) => {
      const track = trackRef.current
      if (track) {
        track.scrollTo({ left: target * track.clientWidth, behavior: reducedMotion ? 'auto' : 'smooth' })
      }
    },
    [reducedMotion],
  )

  const playing =
    !userPaused && shouldAutoplay({ count, seconds: autoplaySeconds, reducedMotion, hovered, focused, hidden })

  useEffect(() => {
    if (!playing) {
      return
    }
    const timer = setTimeout(() => {
      goTo(nextIndex(index, count, 1))
    }, autoplayDelayMs(autoplaySeconds))
    return () => {
      clearTimeout(timer)
    }
  }, [playing, index, count, autoplaySeconds, goTo])

  const onScroll = () => {
    const track = trackRef.current
    if (track) {
      setIndex(indexFromScroll(track.scrollLeft, track.clientWidth, count))
    }
  }

  return (
    <section
      aria-roledescription="carousel"
      aria-label="Featured"
      className="relative"
      onMouseEnter={() => {
        setHovered(true)
      }}
      onMouseLeave={() => {
        setHovered(false)
      }}
      onFocusCapture={() => {
        setFocused(true)
      }}
      onBlurCapture={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) {
          setFocused(false)
        }
      }}
    >
      <div ref={trackRef} onScroll={onScroll} className="scroll-track" aria-live={playing ? 'off' : 'polite'}>
        {children}
      </div>
      {count > 1 ? (
        <>
          <div className="pointer-events-none absolute inset-x-0 bottom-4 flex items-center justify-center gap-3 px-4">
            <div className="pointer-events-auto flex items-center gap-2">
              <button
                type="button"
                aria-label="Previous slide"
                className={CONTROL}
                onClick={() => {
                  goTo(nextIndex(index, count, -1))
                }}
              >
                <Icon name="chevron-left" className="size-5" />
              </button>
              <ul className="flex items-center gap-1 rounded-full bg-primary/70 px-1">
                {Array.from({ length: count }, (_, position) => (
                  <li key={position}>
                    <button
                      type="button"
                      aria-label={`Go to slide ${String(position + 1)}`}
                      aria-current={position === index ? 'true' : undefined}
                      onClick={() => {
                        goTo(position)
                      }}
                      className="group inline-flex size-6 items-center justify-center"
                    >
                      <span className="size-2.5 rounded-full bg-white/50 transition-colors duration-150 ease-out group-aria-[current=true]:bg-white" />
                    </button>
                  </li>
                ))}
              </ul>
              <button
                type="button"
                aria-label="Next slide"
                className={CONTROL}
                onClick={() => {
                  goTo(nextIndex(index, count, 1))
                }}
              >
                <Icon name="chevron-right" className="size-5" />
              </button>
              {autoplaySeconds > 0 && !reducedMotion ? (
                <button
                  type="button"
                  aria-label={userPaused ? 'Start automatic slide show' : 'Pause automatic slide show'}
                  aria-pressed={userPaused}
                  className={CONTROL}
                  onClick={() => {
                    setUserPaused((value) => !value)
                  }}
                >
                  <Icon name={userPaused ? 'play' : 'pause'} className="size-4" />
                </button>
              ) : null}
            </div>
          </div>
        </>
      ) : null}
    </section>
  )
}
