import { useState, type DragEvent } from 'react'
import { useFieldArray, useFormContext } from 'react-hook-form'
import type { PageMedia } from '@/api/pages'
import { Button } from '@/components/ui/Button'
import { EmptyState } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { BLOCK_TYPES, MAX_BLOCKS, blockLabel, createBlock, duplicateBlock, type BlockType } from '@/lib/pageBlocks'
import type { PageFormValues } from '@/lib/pagePayload'
import { BlockCard } from './BlockCard'

export function BlockBuilder({ media }: { media: PageMedia }) {
  const { control, getValues, formState } = useFormContext<PageFormValues>()
  const { fields, append, insert, move, remove } = useFieldArray({ control, name: 'blocks', keyName: 'key' })
  const [menuOpen, setMenuOpen] = useState(false)
  const [openIds, setOpenIds] = useState<ReadonlySet<string>>(new Set())
  const [pendingDelete, setPendingDelete] = useState<number | null>(null)
  const [dragFrom, setDragFrom] = useState<number | null>(null)
  const [dragOver, setDragOver] = useState<number | null>(null)
  const [announcement, setAnnouncement] = useState('')

  const setOpen = (id: string, open: boolean) =>
    setOpenIds((prev) => {
      const next = new Set(prev)
      if (open) next.add(id)
      else next.delete(id)
      return next
    })

  const add = (type: BlockType) => {
    const block = createBlock(type)
    append(block)
    setOpen(block.id, true)
    setMenuOpen(false)
    setAnnouncement(`${blockLabel(type)} block added at position ${fields.length + 1}.`)
  }

  const moveBlock = (from: number, to: number) => {
    move(from, to)
    setAnnouncement(`Block moved to position ${to + 1} of ${fields.length}.`)
  }

  const duplicate = (index: number) => {
    const copy = duplicateBlock(getValues(`blocks.${index}`))
    insert(index + 1, copy)
    setOpen(copy.id, true)
    setAnnouncement(`Block ${index + 1} duplicated.`)
  }

  const confirmDelete = () => {
    if (pendingDelete === null) return
    remove(pendingDelete)
    setAnnouncement(`Block ${pendingDelete + 1} deleted.`)
    setPendingDelete(null)
  }

  const endDrag = () => {
    setDragFrom(null)
    setDragOver(null)
  }

  const onDragStart = (e: DragEvent<HTMLSpanElement>, index: number, id: string) => {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', id)
    const card = e.currentTarget.closest('li')
    if (card) e.dataTransfer.setDragImage(card, 16, 16)
    setDragFrom(index)
  }

  const onDragOver = (e: DragEvent<HTMLLIElement>, index: number) => {
    if (dragFrom === null) return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    if (dragOver !== index) setDragOver(index)
  }

  const onDrop = (e: DragEvent<HTMLLIElement>, index: number) => {
    if (dragFrom === null) return
    e.preventDefault()
    if (dragFrom !== index) moveBlock(dragFrom, index)
    endDrag()
  }

  return (
    <div className="space-y-3">
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
      {fields.length === 0 ? (
        <EmptyState title="This page has no blocks yet.">Use Add block to build the page body.</EmptyState>
      ) : (
        <ol className="space-y-3" aria-label="Page blocks">
          {fields.map((field, index) => {
            const hasErrors = Boolean(formState.errors.blocks?.[index])
            return (
              <BlockCard
                key={field.key}
                index={index}
                type={field.type}
                count={fields.length}
                media={media}
                open={openIds.has(field.id) || hasErrors}
                isDropTarget={dragFrom !== null && dragOver === index && dragFrom !== index}
                onToggle={() => setOpen(field.id, !openIds.has(field.id))}
                onMove={(to) => moveBlock(index, to)}
                onDuplicate={() => duplicate(index)}
                onDelete={() => setPendingDelete(index)}
                onDragStart={(e) => onDragStart(e, index, field.id)}
                onDragEnd={endDrag}
                onDragOver={(e) => onDragOver(e, index)}
                onDrop={(e) => onDrop(e, index)}
              />
            )
          })}
        </ol>
      )}
      <div className="relative inline-block" onKeyDown={(e) => e.key === 'Escape' && setMenuOpen(false)}>
        <Button variant="secondary" aria-expanded={menuOpen} aria-controls="add-block-menu" disabled={fields.length >= MAX_BLOCKS} onClick={() => setMenuOpen((o) => !o)}>
          Add block
        </Button>
        {menuOpen ? (
          <ul id="add-block-menu" aria-label="Block types" className="absolute left-0 z-20 mt-1 w-72 rounded-lg bg-white p-1 shadow-lg ring-1 ring-slate-200">
            {BLOCK_TYPES.map((b) => (
              <li key={b.type}>
                <button type="button" onClick={() => add(b.type)} className="block w-full rounded-md px-3 py-2 text-left hover:bg-slate-100 focus-visible:bg-slate-100">
                  <span className="block text-sm font-medium text-slate-900">{b.label}</span>
                  <span className="block text-xs text-slate-500">{b.description}</span>
                </button>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      {fields.length >= MAX_BLOCKS ? <p className="text-xs text-slate-500">A page can have at most {MAX_BLOCKS} blocks.</p> : null}
      <ConfirmDialog
        open={pendingDelete !== null}
        title="Delete block?"
        message="The block and its content are removed from the page when you save."
        confirmLabel="Delete block"
        danger
        onConfirm={confirmDelete}
        onClose={() => setPendingDelete(null)}
      />
    </div>
  )
}
