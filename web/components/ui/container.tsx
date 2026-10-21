import type { ElementType, ReactNode } from 'react'

interface ContainerProps {
  as?: ElementType
  className?: string
  children: ReactNode
}

export function Container({ as: Tag = 'div', className = '', children }: ContainerProps) {
  return <Tag className={`mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8 ${className}`.trim()}>{children}</Tag>
}
