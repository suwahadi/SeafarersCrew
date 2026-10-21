import type { DragEvent } from 'react'
import { Controller, useFormContext, useWatch } from 'react-hook-form'
import type { BlockType, PageMedia } from '@/api/pages'
import { Button } from '@/components/ui/Button'
import { blockLabel, blockSummary, type FormBlock } from '@/lib/pageBlocks'
import type { PageFormValues } from '@/lib/pagePayload'
import { BlockEditor } from './blocks/BlockEditor'

interface BlockCardProps {
  index: number
  type: BlockType
  count: number
  media: PageMedia
  open: boolean
  isDropTarget: boolean
  onToggle: () => void
  onMove: (to: number) => void
  onDuplicate: () => void
  onDelete: () => void
  onDragStart: (e: DragEvent<HTMLSpanElement>) => void
  onDragEnd: () => void
  onDragOver: (e: DragEvent<HTMLLIElement>) => void
  onDrop: (e: DragEvent<HTMLLIElement>) => void
}

export function BlockCard({ index, type, count, media, open, isDropTarget, onToggle, onMove, onDuplicate, onDelete, onDragStart, onDragEnd, onDragOver, onDrop }: BlockCardProps) {
  const { control } = useFormContext<PageFormValues>()
  const block = useWatch({ control, name: `blocks.${index}` }) as FormBlock | undefined
  const hidden = block?.hidden === true
  const label = blockLabel(type)
  const position = index + 1

  return (
    <li
      aria-label={`Block ${position}: ${label}`}
      role="group"
      onDragOver={onDragOver}
      onDrop={onDrop}
      className={`rounded-lg bg-white ring-1 transition-opacity duration-200 ease-out motion-reduce:transition-none ${isDropTarget ? 'ring-2 ring-navy-500' : 'ring-slate-200'} ${hidden ? 'opacity-60' : ''}`}
      data-hidden={hidden || undefined}
    >
      <div className="flex flex-wrap items-center gap-2 px-3 py-2">
        <span draggable onDragStart={onDragStart} onDragEnd={onDragEnd} title="Drag to reorder" aria-hidden className="cursor-grab select-none px-1 text-slate-400 active:cursor-grabbing">
          ⠿
        </span>
        <button type="button" onClick={onToggle} aria-expanded={open} className="flex min-w-0 flex-1 items-center gap-2 text-left" aria-label={`${open ? 'Collapse' : 'Expand'} block ${position}: ${label}`}>
          <span aria-hidden className="inline-block text-xs text-slate-500 transition-transform duration-150 ease-out motion-reduce:transition-none" style={{ transform: open ? 'rotate(90deg)' : 'none' }}>
            ▶
          </span>
          <span className="shrink-0 rounded bg-navy-50 px-1.5 py-0.5 text-xs font-semibold text-navy-800">{label}</span>
          <span className="truncate text-sm text-slate-600">{block ? blockSummary(block) : ''}</span>
          {hidden ? <span className="shrink-0 rounded bg-slate-200 px-1.5 py-0.5 text-xs text-slate-700">Hidden</span> : null}
        </button>
        <div className="flex items-center gap-1">
          <Button size="sm" variant="ghost" disabled={index === 0} aria-label={`Move block ${position} up`} onClick={() => onMove(index - 1)}>
            ↑
          </Button>
          <Button size="sm" variant="ghost" disabled={index === count - 1} aria-label={`Move block ${position} down`} onClick={() => onMove(index + 1)}>
            ↓
          </Button>
          <Button size="sm" variant="ghost" aria-label={`Duplicate block ${position}`} onClick={onDuplicate}>
            Duplicate
          </Button>
          <Controller
            control={control}
            name={`blocks.${index}.hidden`}
            render={({ field }) => (
              <Button size="sm" variant="ghost" aria-pressed={field.value} aria-label={`${field.value ? 'Show' : 'Hide'} block ${position}`} onClick={() => field.onChange(!field.value)}>
                {field.value ? 'Show' : 'Hide'}
              </Button>
            )}
          />
          <Button size="sm" variant="ghost" aria-label={`Delete block ${position}`} onClick={onDelete}>
            Delete
          </Button>
        </div>
      </div>
      {open ? (
        <div className="border-t border-slate-200 p-4">
          <BlockEditor type={type} index={index} media={media} />
        </div>
      ) : null}
    </li>
  )
}
