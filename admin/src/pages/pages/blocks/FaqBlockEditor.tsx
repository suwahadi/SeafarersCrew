import { useFieldArray, useFormContext } from 'react-hook-form'
import { Button } from '@/components/ui/Button'
import { MAX_FAQ, emptyItem } from '@/lib/pageBlocks'
import type { PageFormValues } from '@/lib/pagePayload'
import { BlockRich, BlockText, ItemRow, ListError, useFieldError, type BlockEditorProps } from './fields'

export function FaqBlockEditor({ index }: BlockEditorProps) {
  const base = `blocks.${index}.settings` as const
  const { control } = useFormContext<PageFormValues>()
  const { fields, append, remove, move } = useFieldArray({ control, name: `${base}.items`, keyName: 'key' })
  const listError = useFieldError(`${base}.items`)

  return (
    <div className="space-y-3">
      <ul className="space-y-3">
        {fields.map((field, i) => (
          <ItemRow key={field.key} label="Question" index={i} count={fields.length} onMove={(to) => move(i, to)} onRemove={() => remove(i)}>
            <BlockText name={`${base}.items.${i}.question`} label="Question" required />
            <BlockRich name={`${base}.items.${i}.answer_html`} label="Answer" minHeight="6rem" />
          </ItemRow>
        ))}
      </ul>
      <ListError message={listError} />
      <Button size="sm" variant="secondary" disabled={fields.length >= MAX_FAQ} onClick={() => append(emptyItem())}>
        Add question
      </Button>
    </div>
  )
}
