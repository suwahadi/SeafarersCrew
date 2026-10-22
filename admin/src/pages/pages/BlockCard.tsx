import { Controller, useFormContext, useWatch } from 'react-hook-form'
import type { BlockType, PageMedia } from '@/api/pages'
import { Button } from '@/components/ui/Button'
import { DragHandle } from '@/components/ui/DragHandle'
import { Icon } from '@/components/ui/Icon'
import type { SortableHandleProps } from '@/hooks/useSortable'
import { blockLabel, blockSummary, type FormBlock } from '@/lib/pageBlocks'
import type { PageFormValues } from '@/lib/pagePayload'
import { BlockEditor } from './blocks/BlockEditor'

interface BlockCardProps {
  index: number
  type: BlockType
  media: PageMedia
  open: boolean
  rowProps: object
  handleProps: SortableHandleProps
  onToggle: () => void
  onDuplicate: () => void
  onDelete: () => void
}

export function BlockCard({ index, type, media, open, rowProps, handleProps, onToggle, onDuplicate, onDelete }: BlockCardProps) {
  const { control } = useFormContext<PageFormValues>()
  const block = useWatch({ control, name: `blocks.${index}` }) as FormBlock | undefined
  const hidden = block?.hidden === true
  const label = blockLabel(type)
  const position = index + 1

  return (
    <li
      aria-label={`Block ${position}: ${label}`}
      role="group"
      {...rowProps}
      className={`rounded-lg bg-white ring-1 ring-slate-200 transition-opacity duration-200 ease-out motion-reduce:transition-none ${hidden ? 'opacity-60' : ''}`}
      data-hidden={hidden || undefined}
    >
      <div className="flex flex-wrap items-center gap-1 px-2 py-2 sm:gap-2 sm:px-3">
        <DragHandle label={`block ${position}`} {...handleProps} />
        <button type="button" onClick={onToggle} aria-expanded={open} className="flex min-h-9 min-w-0 flex-1 basis-32 items-center gap-2 text-left" aria-label={`${open ? 'Collapse' : 'Expand'} block ${position}: ${label}`}>
          <span aria-hidden className="inline-block text-xs text-slate-500 transition-transform duration-150 ease-out motion-reduce:transition-none" style={{ transform: open ? 'rotate(90deg)' : 'none' }}>
            ▶
          </span>
          <span className="shrink-0 rounded bg-navy-50 px-1.5 py-0.5 text-xs font-semibold text-navy-800">{label}</span>
          <span className="truncate text-sm text-slate-600">{block ? blockSummary(block) : ''}</span>
          {hidden ? <span className="shrink-0 rounded bg-slate-200 px-1.5 py-0.5 text-xs text-slate-700">Hidden</span> : null}
        </button>
        <div className="ml-auto flex items-center gap-1">
          <Button size="sm" variant="ghost" icon={<Icon name="copy" />} collapse aria-label={`Duplicate block ${position}`} onClick={onDuplicate}>
            Duplicate
          </Button>
          <Controller
            control={control}
            name={`blocks.${index}.hidden`}
            render={({ field }) => (
              <Button
                size="sm"
                variant="ghost"
                icon={<Icon name={field.value ? 'eye' : 'eyeOff'} />}
                collapse
                aria-pressed={field.value}
                aria-label={`${field.value ? 'Show' : 'Hide'} block ${position}`}
                onClick={() => field.onChange(!field.value)}
              >
                {field.value ? 'Show' : 'Hide'}
              </Button>
            )}
          />
          <Button size="sm" variant="ghost" icon={<Icon name="trash" />} collapse aria-label={`Delete block ${position}`} onClick={onDelete}>
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
