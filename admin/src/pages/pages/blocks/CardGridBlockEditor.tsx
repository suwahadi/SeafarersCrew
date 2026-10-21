import { useFieldArray, useFormContext } from 'react-hook-form'
import { Button } from '@/components/ui/Button'
import { MAX_CARDS, emptyCard } from '@/lib/pageBlocks'
import type { PageFormValues } from '@/lib/pagePayload'
import { BlockImage, BlockSelect, BlockText, BlockTextArea, COLUMN_OPTIONS, ItemRow, ListError, useFieldError, type BlockEditorProps } from './fields'

export function CardGridBlockEditor({ index, media }: BlockEditorProps) {
  const base = `blocks.${index}.settings` as const
  const { control } = useFormContext<PageFormValues>()
  const { fields, append, remove, move } = useFieldArray({ control, name: `${base}.cards`, keyName: 'key' })
  const listError = useFieldError(`${base}.cards`)

  return (
    <div className="space-y-4">
      <div className="max-w-xs">
        <BlockSelect name={`${base}.columns`} label="Columns" options={COLUMN_OPTIONS} numeric />
      </div>
      <ul className="space-y-3">
        {fields.map((field, i) => (
          <ItemRow key={field.key} label="Card" index={i} count={fields.length} onMove={(to) => move(i, to)} onRemove={() => remove(i)}>
            <div className="grid gap-3 md:grid-cols-2">
              <BlockText name={`${base}.cards.${i}.title`} label="Title" required />
              <BlockText name={`${base}.cards.${i}.link`} label="Link" hint="A path like /career, or an https:// address." />
            </div>
            <BlockTextArea name={`${base}.cards.${i}.text`} label="Text" rows={2} />
            <div className="grid gap-3 md:grid-cols-2">
              <BlockText
                name={`${base}.cards.${i}.icon`}
                label="Icon name"
                placeholder="anchor"
                hint="A short name such as ship or anchor: lowercase letters, digits and hyphens."
              />
              <BlockImage name={`${base}.cards.${i}.image_file_id`} label="Image" media={media} hint="Optional. Used instead of the icon." />
            </div>
          </ItemRow>
        ))}
      </ul>
      <ListError message={listError} />
      <Button size="sm" variant="secondary" disabled={fields.length >= MAX_CARDS} onClick={() => append(emptyCard())}>
        Add card
      </Button>
    </div>
  )
}
