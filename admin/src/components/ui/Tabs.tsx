import { useId, useRef, type KeyboardEvent } from 'react'

export interface TabItem {
  id: string
  label: string
  dirty?: boolean
}

export function Tabs({ tabs, active, onChange }: { tabs: TabItem[]; active: string; onChange: (id: string) => void }) {
  const base = useId()
  const refs = useRef<(HTMLButtonElement | null)[]>([])

  const onKey = (e: KeyboardEvent, index: number) => {
    if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return
    e.preventDefault()
    const next = (index + (e.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length
    const tab = tabs[next]
    if (!tab) return
    onChange(tab.id)
    refs.current[next]?.focus()
  }

  return (
    <div role="tablist" aria-orientation="horizontal" className="flex flex-wrap border-b border-slate-200">
      {tabs.map((t, i) => {
        const selected = t.id === active
        return (
          <button
            key={t.id}
            ref={(el) => {
              refs.current[i] = el
            }}
            role="tab"
            type="button"
            id={`${base}-${t.id}`}
            aria-selected={selected}
            tabIndex={selected ? 0 : -1}
            onKeyDown={(e) => onKey(e, i)}
            onClick={() => onChange(t.id)}
            className={`-mb-px whitespace-nowrap border-b-2 px-3 py-2.5 text-sm font-medium transition-colors duration-150 ${
              selected ? 'border-navy-800 text-navy-800' : 'border-transparent text-slate-600 hover:border-slate-300 hover:text-slate-900'
            }`}
          >
            {t.label}
            {t.dirty ? <span aria-label="unsaved changes" className="ml-1 text-orange-600">•</span> : null}
          </button>
        )
      })}
    </div>
  )
}
