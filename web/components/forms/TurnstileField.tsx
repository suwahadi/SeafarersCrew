'use client'

import { useEffect, useRef, useState } from 'react'
import { loadTurnstile, type TurnstileApi } from '@/lib/turnstile'

interface TurnstileFieldProps {
  siteKey: string
  active: boolean
  resetKey: number
  onToken: (token: string) => void
}

const RESERVED_HEIGHT = 'min-h-[65px]'

export function TurnstileField({ siteKey, active, resetKey, onToken }: TurnstileFieldProps) {
  const container = useRef<HTMLDivElement>(null)
  const widget = useRef<{ api: TurnstileApi; id: string } | null>(null)
  const onTokenRef = useRef(onToken)
  const [near, setNear] = useState(false)
  const [failed, setFailed] = useState(false)
  const shouldLoad = siteKey !== '' && (active || near)

  useEffect(() => {
    onTokenRef.current = onToken
  }, [onToken])

  useEffect(() => {
    const node = container.current
    if (siteKey === '' || !node || typeof IntersectionObserver === 'undefined') {
      return undefined
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (window.scrollY > 0 && entries.some((entry) => entry.isIntersecting)) {
          setNear(true)
        }
      },
      { rootMargin: '200px' },
    )
    observer.observe(node)
    return () => {
      observer.disconnect()
    }
  }, [siteKey])

  useEffect(() => {
    const node = container.current
    if (!shouldLoad || !node) {
      return undefined
    }
    let cancelled = false
    loadTurnstile()
      .then((api) => {
        if (cancelled || widget.current) {
          return
        }
        const id = api.render(node, {
          sitekey: siteKey,
          callback: (token) => {
            setFailed(false)
            onTokenRef.current(token)
          },
          'expired-callback': () => {
            onTokenRef.current('')
          },
          'error-callback': () => {
            onTokenRef.current('')
            setFailed(true)
          },
        })
        widget.current = { api, id }
      })
      .catch(() => {
        if (!cancelled) {
          setFailed(true)
        }
      })
    return () => {
      cancelled = true
      if (widget.current) {
        widget.current.api.remove(widget.current.id)
        widget.current = null
      }
    }
  }, [shouldLoad, siteKey])

  useEffect(() => {
    if (resetKey > 0 && widget.current) {
      widget.current.api.reset(widget.current.id)
    }
  }, [resetKey])

  if (siteKey === '') {
    return null
  }
  return (
    <div>
      <div ref={container} data-turnstile className={RESERVED_HEIGHT} />
      {failed ? (
        <p role="alert" className="mt-1.5 text-sm font-medium text-red-700">
          The verification check could not load. Reload the page and try again.
        </p>
      ) : null}
    </div>
  )
}
