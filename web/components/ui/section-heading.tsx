import type { ReactNode } from 'react'

interface SectionHeadingProps {
  title: string
  id?: string
  eyebrow?: string
  action?: ReactNode
  level?: 'h1' | 'h2'
}

export function SectionHeading({ title, id, eyebrow, action, level: Tag = 'h2' }: SectionHeadingProps) {
  return (
    <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
      <div>
        {eyebrow ? <p className="mb-1 text-sm font-semibold uppercase tracking-wider text-accent-text">{eyebrow}</p> : null}
        <Tag id={id} className="text-2xl font-bold tracking-tight text-primary sm:text-3xl">
          {title}
        </Tag>
      </div>
      {action}
    </div>
  )
}
