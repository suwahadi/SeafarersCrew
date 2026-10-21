import { useState, type DragEvent } from 'react'

export function useDragSort(onMove: (fromKey: string, toKey: string) => void) {
  const [dragging, setDragging] = useState<string | null>(null)

  const handleProps = (key: string) => ({
    draggable: true,
    onDragStart: (e: DragEvent) => {
      const transfer = e.dataTransfer as DataTransfer | null
      if (transfer) {
        transfer.effectAllowed = 'move'
        transfer.setData('text/plain', key)
      }
      setDragging(key)
    },
    onDragEnd: () => setDragging(null),
  })

  const targetProps = (key: string) => ({
    onDragOver: (e: DragEvent) => {
      if (dragging !== null) {
        e.preventDefault()
        e.stopPropagation()
      }
    },
    onDrop: (e: DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (dragging !== null && dragging !== key) onMove(dragging, key)
      setDragging(null)
    },
  })

  return { dragging, handleProps, targetProps }
}
