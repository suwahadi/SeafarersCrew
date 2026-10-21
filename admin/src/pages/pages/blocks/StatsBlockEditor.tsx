import { useFieldArray, useFormContext } from 'react-hook-form'
import { Button } from '@/components/ui/Button'
import { MAX_STATS, emptyItem } from '@/lib/pageBlocks'
import type { PageFormValues } from '@/lib/pagePayload'
import { BlockText, ItemRow, ListError, useFieldError, type BlockEditorProps } from './fields'

export function StatsBlockEditor({ index }: BlockEditorProps) {
  const base = `blocks.${index}.settings` as const
  const { control } = useFormContext<PageFormValues>()
  const { fields, append, remove, move } = useFieldArray({ control, name: `${base}.items`, keyName: 'key' })
  const listError = useFieldError(`${base}.items`)

  return (
    <div className="space-y-3">
      <ul className="space-y-3">
        {fields.map((field, i) => (
          <ItemRow key={field.key} label="Stat" index={i} count={fields.length} onMove={(to) => move(i, to)} onRemove={() => remove(i)}>
            <div className="grid gap-3 md:grid-cols-[8rem_8rem_1fr]">
              <BlockText name={`${base}.items.${i}.number`} label="Number" type="number" numeric min={0} />
              <BlockText name={`${base}.items.${i}.suffix`} label="Suffix" placeholder="+" />
              <BlockText name={`${base}.items.${i}.label`} label="Label" required />
            </div>
          </ItemRow>
        ))}
      </ul>
      <ListError message={listError} />
      <Button size="sm" variant="secondary" disabled={fields.length >= MAX_STATS} onClick={() => append(emptyItem())}>
        Add stat
      </Button>
    </div>
  )
}
