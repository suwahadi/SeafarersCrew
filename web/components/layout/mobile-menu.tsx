'use client'

import { usePathname } from 'next/navigation'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Icon } from '@/components/ui/icons'

export function MobileMenu({ children }: { children: ReactNode }) {
  const pathname = usePathname()
  const [openAt, setOpenAt] = useState<string | null>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const open = openAt === pathname

  useEffect(() => {
    if (!open) {
      return
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpenAt(null)
        buttonRef.current?.focus()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  return (
    <div className="lg:hidden">
      <button
        ref={buttonRef}
        type="button"
        aria-expanded={open}
        aria-controls="mobile-navigation"
        aria-label={open ? 'Close menu' : 'Open menu'}
        onClick={() => {
          setOpenAt(open ? null : pathname)
        }}
        className="inline-flex size-10 items-center justify-center rounded-md border border-line text-primary transition-colors duration-150 ease-out hover:bg-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-text"
      >
        <Icon name={open ? 'close' : 'menu'} className="size-5" />
      </button>
      <div
        id="mobile-navigation"
        hidden={!open}
        className="absolute inset-x-0 top-full max-h-[calc(100dvh-4.5rem)] overflow-y-auto border-t border-line bg-white shadow-lg"
      >
        {children}
      </div>
    </div>
  )
}
