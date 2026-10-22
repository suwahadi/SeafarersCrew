import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Spinner } from './Spinner'

type Variant = 'primary' | 'secondary' | 'danger' | 'ghost'
type Size = 'sm' | 'md'

const variants: Record<Variant, string> = {
  primary: 'bg-navy-800 text-white hover:bg-navy-700 disabled:bg-navy-200',
  secondary: 'bg-white text-slate-800 ring-1 ring-inset ring-slate-300 hover:bg-slate-50 disabled:text-slate-400',
  danger: 'bg-red-700 text-white hover:bg-red-800 disabled:bg-red-300',
  ghost: 'text-slate-700 hover:bg-slate-100 disabled:text-slate-400',
}

const sizes: Record<Size, string> = {
  sm: 'h-8 px-3 text-sm',
  md: 'h-9 px-4 text-sm',
}

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
  icon?: ReactNode
  collapse?: boolean
}

export function Button({ variant = 'primary', size = 'md', loading = false, icon, collapse = false, children, className = '', disabled, type = 'button', ...rest }: ButtonProps) {
  const collapsed = collapse && icon !== undefined
  const label = typeof children === 'string' ? children : undefined
  return (
    <button
      type={type}
      disabled={disabled ?? loading}
      aria-busy={loading || undefined}
      aria-label={collapsed ? (rest['aria-label'] ?? label) : rest['aria-label']}
      title={collapsed ? (rest.title ?? label) : rest.title}
      className={`inline-flex items-center justify-center gap-2 rounded-md font-medium transition-colors duration-150 ease-out disabled:cursor-not-allowed ${variants[variant]} ${sizes[size]} ${collapsed ? 'max-sm:size-9 max-sm:shrink-0 max-sm:px-0' : ''} ${className}`}
      {...rest}
    >
      {loading ? <Spinner size="sm" /> : icon}
      {collapsed ? <span className="max-sm:sr-only">{children}</span> : children}
    </button>
  )
}
