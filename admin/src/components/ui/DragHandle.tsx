import type { ButtonHTMLAttributes } from 'react'
import { Icon } from './Icon'

export function DragHandle({ label, className = '', ...rest }: { label: string } & ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      type="button"
      data-sort-handle=""
      aria-label={`Reorder ${label}. Drag, or press the up and down arrow keys.`}
      title="Drag to reorder"
      className={`inline-flex size-9 shrink-0 cursor-grab touch-none items-center justify-center rounded-md text-slate-400 select-none hover:bg-slate-100 hover:text-slate-700 active:cursor-grabbing ${className}`}
      {...rest}
    >
      <Icon name="grip" className="size-5" strokeWidth={3} />
    </button>
  )
}
