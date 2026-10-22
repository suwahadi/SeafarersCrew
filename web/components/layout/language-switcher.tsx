'use client'

import { useEffect, useRef, useState } from 'react'
import { Icon } from '@/components/ui/icons'
import { flagSrc } from '@/lib/flags'
import { expiredTranslateCookies, isSourceLanguage, translateCookie } from '@/lib/language'
import type { Language } from '@/lib/types'

const SCRIPT_ID = 'google-translate-script'
const ELEMENT_ID = 'google_translate_element'
const CALLBACK = 'googleTranslateElementInit'

type TranslateElementConstructor = new (options: { pageLanguage: string; autoDisplay: boolean }, elementId: string) => unknown

declare global {
  interface Window {
    googleTranslateElementInit?: () => void
    google?: { translate?: { TranslateElement?: TranslateElementConstructor } }
  }
}

interface LanguageSwitcherProps {
  languages: Language[]
  sourceCode: string
  initialCode: string
}

function loadTranslateWidget(sourceCode: string, onFailure: () => void): void {
  if (document.getElementById(SCRIPT_ID)) {
    return
  }
  window[CALLBACK] = () => {
    const Constructor = window.google?.translate?.TranslateElement
    if (Constructor) {
      new Constructor({ pageLanguage: sourceCode, autoDisplay: false }, ELEMENT_ID)
    }
  }
  const script = document.createElement('script')
  script.id = SCRIPT_ID
  script.async = true
  script.src = `https://translate.google.com/translate_a/element.js?cb=${CALLBACK}`
  script.onerror = onFailure
  document.body.appendChild(script)
}

function storeTranslateCookie(code: string, sourceCode: string): void {
  document.cookie = translateCookie(code, sourceCode)
}

function clearTranslateCookies(): void {
  for (const cookie of expiredTranslateCookies(window.location.hostname)) {
    document.cookie = cookie
  }
}

function Flag({ language }: { language: Language }) {
  const src = flagSrc(language)
  if (src) {
    return (
      <img
        src={src}
        alt=""
        width={24}
        height={16}
        className="h-4 w-6 rounded-[2px] object-cover shadow-[0_0_0_1px_rgb(0_0_0/0.15)]"
      />
    )
  }
  return (
    <span aria-hidden="true" className="inline-flex h-4 w-6 items-center justify-center rounded-[2px] bg-surface text-[10px] font-bold uppercase text-primary">
      {language.code.slice(0, 2)}
    </span>
  )
}

export function LanguageSwitcher({ languages, sourceCode, initialCode }: LanguageSwitcherProps) {
  const [activeCode, setActiveCode] = useState(initialCode)
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (isSourceLanguage(initialCode, sourceCode)) {
      return
    }
    loadTranslateWidget(sourceCode, () => {
      clearTranslateCookies()
      setActiveCode(sourceCode)
    })
  }, [initialCode, sourceCode])

  useEffect(() => {
    if (!open) {
      return
    }
    const onPointerDown = (event: PointerEvent) => {
      if (rootRef.current && event.target instanceof Node && !rootRef.current.contains(event.target)) {
        setOpen(false)
      }
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
        buttonRef.current?.focus()
      }
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  const active = languages.find((language) => language.code === activeCode) ?? languages.find((language) => language.is_source) ?? languages[0]

  const select = (code: string) => {
    setOpen(false)
    if (code === activeCode) {
      return
    }
    if (isSourceLanguage(code, sourceCode)) {
      clearTranslateCookies()
      window.location.reload()
      return
    }
    storeTranslateCookie(code, sourceCode)
    if (document.getElementById(SCRIPT_ID)) {
      window.location.reload()
      return
    }
    setActiveCode(code)
    loadTranslateWidget(sourceCode, () => {
      clearTranslateCookies()
      setActiveCode(sourceCode)
    })
  }

  if (!active || languages.length < 2) {
    return null
  }

  return (
    <div ref={rootRef} className="relative" translate="no">
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="true"
        aria-expanded={open}
        aria-controls="language-menu"
        aria-label={`Language: ${active.name}`}
        onClick={() => {
          setOpen((value) => !value)
        }}
        className="inline-flex h-10 items-center justify-center gap-1.5 rounded-md border border-line px-2.5 text-primary transition-colors duration-150 ease-out hover:bg-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-text"
      >
        <Flag language={active} />
        <Icon name="chevron-down" className="size-4" />
      </button>
      <ul
        id="language-menu"
        hidden={!open}
        className="absolute right-0 z-50 mt-2 max-h-80 w-56 overflow-auto rounded-md border border-line bg-white p-1 shadow-lg"
      >
        {languages.map((language) => (
          <li key={language.code}>
            <button
              type="button"
              lang={language.code}
              aria-current={language.code === active.code ? 'true' : undefined}
              onClick={() => {
                select(language.code)
              }}
              className="flex w-full items-center gap-2 rounded px-3 py-2 text-left text-sm text-ink hover:bg-surface focus-visible:bg-surface aria-[current=true]:font-semibold"
            >
              <Flag language={language} />
              {language.name}
            </button>
          </li>
        ))}
      </ul>
      <div id={ELEMENT_ID} className="hidden" />
    </div>
  )
}
