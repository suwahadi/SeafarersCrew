import Link from 'next/link'
import type { ReactNode } from 'react'
import { isInternalHref, safeHref } from '@/lib/links'

interface SmartLinkProps {
  href: string
  newTab?: boolean
  className?: string
  children: ReactNode
  ariaLabel?: string
}

export function SmartLink({ href, newTab = false, className, children, ariaLabel }: SmartLinkProps) {
  const target = safeHref(href)
  if (target === '') {
    return <span className={className}>{children}</span>
  }
  if (isInternalHref(target) && !newTab) {
    return (
      <Link href={target} className={className} aria-label={ariaLabel}>
        {children}
      </Link>
    )
  }
  return (
    <a
      href={target}
      className={className}
      aria-label={ariaLabel}
      {...(newTab ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
    >
      {children}
    </a>
  )
}
