'use client'

import { useEffect, useRef } from 'react'
import { valueAt } from '@/lib/count'
import { formatNumber } from '@/lib/format'

const DURATION_MS = 1200
const STEP_MS = 40

interface CountUpProps {
  value: number
  suffix?: string
  className?: string
}

export function CountUp({ value, suffix = '', className }: CountUpProps) {
  const ref = useRef<HTMLSpanElement>(null)

  useEffect(() => {
    const textNode = ref.current?.firstChild
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (!textNode || reduced || value <= 0 || typeof IntersectionObserver === 'undefined') {
      return
    }
    const final = formatNumber(value)
    let timer: ReturnType<typeof setInterval> | undefined
    textNode.nodeValue = formatNumber(0)

    const observer = new IntersectionObserver((entries) => {
      if (!entries.some((entry) => entry.isIntersecting)) {
        return
      }
      observer.disconnect()
      const startedAt = performance.now()
      timer = setInterval(() => {
        const elapsed = performance.now() - startedAt
        textNode.nodeValue = formatNumber(valueAt(value, elapsed, DURATION_MS))
        if (elapsed >= DURATION_MS) {
          clearInterval(timer)
        }
      }, STEP_MS)
    })
    if (ref.current) {
      observer.observe(ref.current)
    }

    return () => {
      observer.disconnect()
      clearInterval(timer)
      textNode.nodeValue = final
    }
  }, [value])

  return (
    <span className={className} translate="no">
      <span ref={ref} className="count-value">
        {formatNumber(value)}
      </span>
      {suffix}
    </span>
  )
}
