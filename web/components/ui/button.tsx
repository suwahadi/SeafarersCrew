import type { ReactNode } from 'react'
import { SmartLink } from '@/components/ui/smart-link'

type Variant = 'primary' | 'accent' | 'outline' | 'light'

const BASE =
  'inline-flex items-center justify-center gap-2 rounded-md px-5 py-3 text-sm font-semibold transition-colors duration-200 ease-out focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-text'

const VARIANTS: Record<Variant, string> = {
  primary: 'bg-primary text-on-primary hover:bg-primary-hover',
  accent: 'bg-accent text-on-accent hover:bg-accent-hover',
  outline: 'border border-primary text-primary hover:bg-primary hover:text-on-primary',
  light: 'bg-white text-primary hover:bg-slate-100',
}

export function buttonClass(variant: Variant = 'primary', extra = ''): string {
  return `${BASE} ${VARIANTS[variant]} ${extra}`.trim()
}

interface ButtonLinkProps {
  href: string
  variant?: Variant
  newTab?: boolean
  className?: string
  children: ReactNode
}

export function ButtonLink({ href, variant = 'primary', newTab, className, children }: ButtonLinkProps) {
  return (
    <SmartLink href={href} newTab={newTab} className={buttonClass(variant, className)}>
      {children}
    </SmartLink>
  )
}
