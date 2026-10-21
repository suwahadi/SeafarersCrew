'use client'

import { useEffect, useState } from 'react'
import { Icon } from '@/components/ui/icons'

const RESET_MS = 2000

export function CopyLinkButton({ url, className }: { url: string; className?: string }) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) {
      return
    }
    const timer = setTimeout(() => {
      setCopied(false)
    }, RESET_MS)
    return () => {
      clearTimeout(timer)
    }
  }, [copied])

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
    } catch {
      setCopied(false)
    }
  }

  return (
    <button type="button" onClick={() => void copy()} className={className}>
      <Icon name={copied ? 'check' : 'link'} className="size-4" />
      <span aria-live="polite">{copied ? 'Link copied' : 'Copy link'}</span>
    </button>
  )
}
