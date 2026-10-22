import { useCallback, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react'

const EDGE = 64
const STEP = 14

export interface SortableHandleProps {
  onPointerDown: (e: PointerEvent<HTMLElement>) => void
  onPointerMove: (e: PointerEvent<HTMLElement>) => void
  onPointerUp: (e: PointerEvent<HTMLElement>) => void
  onPointerCancel: (e: PointerEvent<HTMLElement>) => void
  onKeyDown: (e: KeyboardEvent<HTMLElement>) => void
  onClick: (e: React.MouseEvent<HTMLElement>) => void
}

export function useSortable(keys: readonly string[], onMove: (fromKey: string, toKey: string) => void) {
  const [dragging, setDragging] = useState<string | null>(null)
  const [over, setOver] = useState<string | null>(null)
  const scope = useRef<string>(`sortable-${Math.random().toString(36).slice(2)}`)
  const last = useRef<string | null>(null)

  const targetAt = useCallback((x: number, y: number): string | null => {
    for (const el of document.elementsFromPoint(x, y)) {
      const host = el.closest<HTMLElement>(`[data-sort-scope="${scope.current}"]`)
      if (host?.dataset.sortKey) return host.dataset.sortKey
    }
    return null
  }, [])

  const rowProps = (key: string) => ({
    'data-sort-scope': scope.current,
    'data-sort-key': key,
    'data-sort-state': dragging === key ? 'dragging' : over === key && dragging !== null ? 'over' : undefined,
  })

  const handleProps = (key: string): SortableHandleProps => ({
    onPointerDown: (e) => {
      if (e.button !== 0 && e.pointerType === 'mouse') return
      e.currentTarget.setPointerCapture(e.pointerId)
      last.current = key
      setDragging(key)
      setOver(null)
    },
    onPointerMove: (e) => {
      if (dragging !== key) return
      if (e.clientY < EDGE) window.scrollBy(0, -STEP)
      else if (e.clientY > window.innerHeight - EDGE) window.scrollBy(0, STEP)
      const target = targetAt(e.clientX, e.clientY)
      setOver(target !== null && target !== key ? target : null)
    },
    onPointerUp: (e) => {
      if (dragging !== key) return
      e.currentTarget.releasePointerCapture(e.pointerId)
      const target = targetAt(e.clientX, e.clientY)
      setDragging(null)
      setOver(null)
      if (target !== null && target !== key) onMove(key, target)
    },
    onPointerCancel: () => {
      setDragging(null)
      setOver(null)
    },
    onKeyDown: (e) => {
      if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') return
      e.preventDefault()
      const i = keys.indexOf(key)
      const next = keys[e.key === 'ArrowUp' ? i - 1 : i + 1]
      if (next === undefined) return
      onMove(key, next)
      requestAnimationFrame(() => document.querySelector<HTMLElement>(`[data-sort-scope="${scope.current}"][data-sort-key="${key}"] [data-sort-handle]`)?.focus())
    },
    onClick: (e) => e.stopPropagation(),
  })

  return { dragging, over, rowProps, handleProps }
}
