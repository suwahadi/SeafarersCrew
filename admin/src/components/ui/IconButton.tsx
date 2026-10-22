import type { ButtonHTMLAttributes } from 'react'
import { Icon, type IconName } from './Icon'

type Tone = 'neutral' | 'danger' | 'primary'

const tones: Record<Tone, string> = {
  neutral: 'text-slate-600 hover:bg-slate-100 hover:text-slate-900 disabled:text-slate-300',
  danger: 'text-red-700 hover:bg-red-50 disabled:text-red-200',
  primary: 'bg-navy-800 text-white hover:bg-navy-700 disabled:bg-navy-200',
}

export interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  icon: IconName
  label: string
  tone?: Tone
  bordered?: boolean
}

export function IconButton({ icon, label, tone = 'neutral', bordered = false, className = '', type = 'button', ...rest }: IconButtonProps) {
  return (
    <button
      type={type}
      aria-label={label}
      title={label}
      className={`inline-flex size-9 shrink-0 items-center justify-center rounded-md transition-colors duration-150 ease-out disabled:cursor-not-allowed ${bordered ? 'ring-1 ring-inset ring-slate-300 bg-white' : ''} ${tones[tone]} ${className}`}
      {...rest}
    >
      <Icon name={icon} className="size-[1.125rem]" />
    </button>
  )
}
