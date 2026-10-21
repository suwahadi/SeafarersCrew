'use client'

import { useId, useState } from 'react'
import { Icon } from '@/components/ui/icons'
import { RichHtml } from '@/components/ui/rich-html'

export interface FaqEntry {
  question: string
  answerHtml: string
}

export function FaqAccordion({ items }: { items: readonly FaqEntry[] }) {
  const baseId = useId()
  const [openIndexes, setOpenIndexes] = useState<ReadonlySet<number>>(new Set())

  const toggle = (index: number) => {
    setOpenIndexes((current) => {
      const next = new Set(current)
      if (!next.delete(index)) {
        next.add(index)
      }
      return next
    })
  }

  return (
    <div className="divide-y divide-line rounded-lg border border-line bg-white">
      {items.map((item, index) => {
        const open = openIndexes.has(index)
        const buttonId = `${baseId}-q-${String(index)}`
        const panelId = `${baseId}-a-${String(index)}`
        return (
          <div key={buttonId}>
            <h2>
              <button
                type="button"
                id={buttonId}
                aria-expanded={open}
                aria-controls={panelId}
                onClick={() => {
                  toggle(index)
                }}
                className="flex w-full items-center justify-between gap-4 px-5 py-4 text-left text-base font-semibold text-primary transition-colors duration-150 ease-out hover:bg-surface"
              >
                {item.question}
                <Icon name="chevron-down" className="faq-chevron size-5 shrink-0" />
              </button>
            </h2>
            <div id={panelId} role="region" aria-labelledby={buttonId} data-open={open} className="faq-panel">
              <div inert={!open}>
                <RichHtml html={item.answerHtml} className="prose-wide px-5 pb-5" />
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
