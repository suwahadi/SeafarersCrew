import { useQuery } from '@tanstack/react-query'
import { Controller, useFormContext } from 'react-hook-form'
import { errorMessage } from '@/api/client'
import type { AdminImage } from '@/api/cmsMedia'
import { galleryOptionsApi, type GalleryOption } from '@/api/galleryOptions'
import { DragHandle } from '@/components/ui/DragHandle'
import { IconButton } from '@/components/ui/IconButton'
import { Spinner } from '@/components/ui/Spinner'
import { useSortable } from '@/hooks/useSortable'
import { MAX_GALLERY_IMAGES, moveItem } from '@/lib/pageBlocks'
import type { PageFormValues } from '@/lib/pagePayload'
import { BlockCheckbox, BlockSelect, COLUMN_OPTIONS, ListError, useFieldError, type BlockEditorProps } from './fields'

export const galleryOptionsKey = ['gallery-options'] as const

function imageSrc(image: AdminImage | null | undefined): string | undefined {
  if (!image) return undefined
  return image.thumb_url || image.url
}

function Thumb({ option }: { option: GalleryOption | undefined }) {
  const src = imageSrc(option?.image)
  return (
    <span className="flex h-12 w-16 shrink-0 items-center justify-center overflow-hidden rounded bg-slate-100">
      {src ? <img src={src} alt="" width={64} height={48} className="h-full w-full object-cover" /> : null}
    </span>
  )
}

function optionName(option: GalleryOption | undefined): string {
  return option ? option.caption || option.alt_text : 'Image no longer available'
}

function SelectedImages({ selected, byId, onChange, onRemove }: { selected: string[]; byId: Map<string, GalleryOption>; onChange: (ids: string[]) => void; onRemove: (id: string) => void }) {
  const sort = useSortable(selected, (from, to) => onChange(moveItem(selected, selected.indexOf(from), selected.indexOf(to))))
  return (
    <ol className="space-y-1">
      {selected.map((id, i) => {
        const option = byId.get(id)
        const name = optionName(option)
        return (
          <li key={id} {...sort.rowProps(id)} className="flex items-center gap-1 rounded-md bg-slate-50 p-1.5 ring-1 ring-slate-200 sm:gap-2">
            <DragHandle label={name} {...sort.handleProps(id)} />
            <Thumb option={option} />
            <span className="min-w-0 flex-1 truncate text-sm text-slate-800">
              {i + 1}. {name}
            </span>
            <IconButton icon="trash" tone="danger" label={`Remove ${name}`} onClick={() => onRemove(id)} />
          </li>
        )
      })}
    </ol>
  )
}

export function GalleryBlockEditor({ index }: BlockEditorProps) {
  const base = `blocks.${index}.settings` as const
  const { control } = useFormContext<PageFormValues>()
  const options = useQuery({ queryKey: galleryOptionsKey, queryFn: galleryOptionsApi.active })
  const error = useFieldError(`${base}.image_ids`)
  const byId = new Map((options.data ?? []).map((o) => [o.id, o]))

  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2">
        <BlockSelect name={`${base}.columns`} label="Columns" options={COLUMN_OPTIONS} numeric />
        <div className="self-end pb-2">
          <BlockCheckbox name={`${base}.lightbox`} label="Open images in a lightbox" />
        </div>
      </div>
      <Controller
        control={control}
        name={`${base}.image_ids`}
        render={({ field }) => {
          const selected = field.value
          const toggle = (id: string) => field.onChange(selected.includes(id) ? selected.filter((s) => s !== id) : [...selected, id])
          return (
            <div className="space-y-4">
              <div>
                <p className="mb-1 text-sm font-medium text-slate-700">Pick images</p>
                {options.isPending ? (
                  <Spinner label="Loading gallery" />
                ) : options.isError ? (
                  <p role="alert" className="text-sm text-red-700">
                    {errorMessage(options.error)}
                  </p>
                ) : options.data.length === 0 ? (
                  <p className="text-sm text-slate-500">The gallery has no active images yet. Add some in Website › Gallery.</p>
                ) : (
                  <ul className="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-6" aria-label="Gallery images">
                    {options.data.map((option) => {
                      const position = selected.indexOf(option.id)
                      const isSelected = position >= 0
                      return (
                        <li key={option.id}>
                          <button
                            type="button"
                            aria-pressed={isSelected}
                            aria-label={optionName(option)}
                            disabled={!isSelected && selected.length >= MAX_GALLERY_IMAGES}
                            onClick={() => toggle(option.id)}
                            className={`relative block w-full overflow-hidden rounded-md ring-2 transition-opacity duration-150 ease-out motion-reduce:transition-none disabled:opacity-40 ${isSelected ? 'ring-navy-600' : 'ring-transparent hover:ring-slate-300'}`}
                            style={{ aspectRatio: '4 / 3' }}
                          >
                            {option.image ? <img src={imageSrc(option.image)} alt="" width={160} height={120} className="h-full w-full object-cover" /> : null}
                            {isSelected ? (
                              <span aria-hidden className="absolute top-1 left-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-navy-800 px-1 text-xs font-semibold text-white">
                                {position + 1}
                              </span>
                            ) : null}
                          </button>
                        </li>
                      )
                    })}
                  </ul>
                )}
              </div>
              <div>
                <p className="mb-1 text-sm font-medium text-slate-700">
                  Selected images ({selected.length}/{MAX_GALLERY_IMAGES})
                </p>
                {selected.length === 0 ? (
                  <p className="text-sm text-slate-500">Nothing selected yet. The order here is the order on the page.</p>
                ) : (
                  <SelectedImages selected={selected} byId={byId} onChange={field.onChange} onRemove={toggle} />
                )}
              </div>
            </div>
          )
        }}
      />
      <ListError message={error} />
    </div>
  )
}
