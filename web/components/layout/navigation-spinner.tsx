'use client'

import { usePathname, useSearchParams } from 'next/navigation'
import { useEffect, useState } from 'react'
import { isInternalNavigation } from '@/lib/navigation'

const GIVE_UP_MS = 15_000

export function NavigationSpinner() {
  const pathname = usePathname()
  const search = useSearchParams().toString()
  const location = `${pathname}?${search}`
  const [startedAt, setStartedAt] = useState<string | null>(null)

  useEffect(() => {
    const onClick = (event: MouseEvent) => {
      const anchor = event.target instanceof Element ? event.target.closest('a') : null
      if (anchor && isInternalNavigation(event, anchor, window.location)) {
        setStartedAt(location)
      }
    }
    document.addEventListener('click', onClick)
    return () => {
      document.removeEventListener('click', onClick)
    }
  }, [location])

  useEffect(() => {
    if (startedAt === null) {
      return
    }
    const timer = setTimeout(() => {
      setStartedAt(null)
    }, GIVE_UP_MS)
    return () => {
      clearTimeout(timer)
    }
  }, [startedAt])

  if (startedAt !== location) {
    return null
  }
  return (
    <div role="status" className="route-loading">
      <span className="spinner" />
      <span className="sr-only">Loading</span>
    </div>
  )
}
