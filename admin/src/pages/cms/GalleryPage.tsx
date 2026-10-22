import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { z } from 'zod'
import type { GalleryItem } from '@/api/cms'
import { ImageField } from '@/components/cms/ImageField'
import { Thumb } from '@/components/cms/Thumb'
import { Button } from '@/components/ui/Button'
import { CheckboxField, TextAreaField, TextField } from '@/components/ui/Field'
import { Badge } from '@/components/ui/Layout'
import { CmsFormFrame, useCmsSave } from './cmsForm'
import { CmsListPage, type CmsConfig, type CmsFormProps } from './CmsListPage'
import { GalleryBulkUpload } from './GalleryBulkUpload'

const schema = z.object({
  image_file_id: z.string().min(1, 'Upload the image.'),
  alt_text: z.string().trim().min(1, 'Describe the image for screen readers.').max(200, 'The alt text may not exceed 200 characters.'),
  caption: z.string().trim().max(300, 'The caption may not exceed 300 characters.'),
  show_on_home: z.boolean(),
  is_active: z.boolean(),
})

type Values = z.infer<typeof schema>

const FIELDS = ['image_file_id', 'alt_text', 'caption', 'show_on_home', 'is_active'] as const

export function GalleryForm({ row, onDone }: CmsFormProps<GalleryItem>) {
  const { register, control, handleSubmit, setError, formState } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      image_file_id: row?.image?.file_id ?? '',
      alt_text: row?.alt_text ?? '',
      caption: row?.caption ?? '',
      show_on_home: row?.show_on_home ?? false,
      is_active: row?.is_active ?? true,
    },
  })
  const { mutation, formError } = useCmsSave<GalleryItem, Values>({ segment: 'gallery', row, fields: FIELDS, setError, toBody: (v) => ({ ...v }), onDone, label: 'Photo' })
  const errors = formState.errors

  return (
    <CmsFormFrame onSubmit={handleSubmit((v) => mutation.mutate(v))} onCancel={onDone} pending={mutation.isPending} formError={formError}>
      <Controller
        control={control}
        name="image_file_id"
        render={({ field }) => <ImageField label="Image" required value={field.value} onChange={field.onChange} image={row?.image} aspect="4 / 3" error={errors.image_file_id?.message} />}
      />
      <TextField label="Alt text" required maxLength={200} hint="Describe what the image shows." error={errors.alt_text?.message} {...register('alt_text')} />
      <TextAreaField label="Caption" maxLength={300} error={errors.caption?.message} {...register('caption')} />
      <div className="flex flex-wrap gap-6">
        <CheckboxField label="Show on home page" {...register('show_on_home')} />
        <CheckboxField label="Active" {...register('is_active')} />
      </div>
    </CmsFormFrame>
  )
}

export const galleryConfig: CmsConfig<GalleryItem> = {
  segment: 'gallery',
  title: 'Gallery',
  singular: 'Photo',
  description: 'Photos shown in the website gallery. Drag to set the display order.',
  layout: 'grid',
  label: (r) => r.alt_text || r.caption || 'Photo',
  summary: (r) => (
    <div className="flex flex-col gap-2">
      <Thumb image={r.image} width={200} height={150} alt={r.alt_text} fit="cover" fluid />
      <div className="min-w-0">
        <p className="truncate text-sm text-slate-900">{r.caption || r.alt_text}</p>
        <p className="mt-1 flex gap-1">{r.show_on_home ? <Badge tone="blue">Home</Badge> : null}</p>
      </div>
    </div>
  ),
  toPayload: (r, isActive) => ({ image_file_id: r.image?.file_id ?? '', caption: r.caption, alt_text: r.alt_text, show_on_home: r.show_on_home, is_active: isActive }),
  Form: GalleryForm,
  tabs: [
    { id: 'home', label: 'On home page', match: (r) => r.show_on_home },
    { id: 'inactive', label: 'Inactive', match: (r) => !r.is_active },
  ],
  emptyHint: 'Upload photos with the bulk uploader or add them one at a time.',
}

export default function GalleryPage() {
  const [bulkOpen, setBulkOpen] = useState(false)
  return (
    <>
      <CmsListPage
        config={galleryConfig}
        headerActions={
          <Button variant="secondary" onClick={() => setBulkOpen(true)}>
            Bulk upload
          </Button>
        }
      />
      <GalleryBulkUpload open={bulkOpen} onClose={() => setBulkOpen(false)} />
    </>
  )
}
