import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import type { Testimonial } from '@/api/cms'
import { ImageField } from '@/components/cms/ImageField'
import { RatingDisplay, RatingInput } from '@/components/cms/Rating'
import { Thumb } from '@/components/cms/Thumb'
import { CheckboxField, TextAreaField, TextField } from '@/components/ui/Field'
import { CmsFormFrame, nullable, useCmsSave } from './cmsForm'
import { CmsListPage, type CmsConfig, type CmsFormProps } from './CmsListPage'

const QUOTE_MAX = 1000

const schema = z.object({
  name: z.string().trim().min(1, 'This field is required.').max(120, 'The name may not exceed 120 characters.'),
  position: z.string().trim().max(160, 'The position may not exceed 160 characters.'),
  photo_file_id: z.string(),
  quote: z.string().trim().min(1, 'This field is required.').max(QUOTE_MAX, `The quote may not exceed ${QUOTE_MAX} characters.`),
  rating: z.number().int().min(1, 'Choose a rating from 1 to 5.').max(5),
  is_active: z.boolean(),
})

type Values = z.infer<typeof schema>

const FIELDS = ['name', 'position', 'photo_file_id', 'quote', 'rating', 'is_active'] as const

function body(v: Values): Record<string, unknown> {
  return { name: v.name, position: v.position, photo_file_id: nullable(v.photo_file_id), quote: v.quote, rating: v.rating, is_active: v.is_active }
}

export function TestimonialForm({ row, onDone }: CmsFormProps<Testimonial>) {
  const { register, control, handleSubmit, setError, formState } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: row?.name ?? '',
      position: row?.position ?? '',
      photo_file_id: row?.photo?.file_id ?? '',
      quote: row?.quote ?? '',
      rating: row?.rating ?? 0,
      is_active: row?.is_active ?? true,
    },
  })
  const { mutation, formError } = useCmsSave<Testimonial, Values>({ segment: 'testimonials', row, fields: FIELDS, setError, toBody: body, onDone, label: 'Testimonial' })
  const quote = useWatch({ control, name: 'quote' })
  const errors = formState.errors

  return (
    <CmsFormFrame onSubmit={handleSubmit((v) => mutation.mutate(v))} onCancel={onDone} pending={mutation.isPending} formError={formError}>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <TextField label="Name" required maxLength={120} error={errors.name?.message} {...register('name')} />
        <TextField label="Position" maxLength={160} hint="For example, Chief Officer, MV Ocean Star." error={errors.position?.message} {...register('position')} />
      </div>
      <Controller
        control={control}
        name="photo_file_id"
        render={({ field }) => <ImageField label="Photo" value={field.value} onChange={field.onChange} image={row?.photo} aspect="1 / 1" hint="Optional square photo." error={errors.photo_file_id?.message} />}
      />
      <div>
        <TextAreaField label="Quote" required rows={5} error={errors.quote?.message} {...register('quote')} />
        <p className={`mt-1 text-right text-xs ${quote.length > QUOTE_MAX ? 'text-red-600' : 'text-slate-500'}`} aria-live="polite">
          {quote.length} / {QUOTE_MAX}
        </p>
      </div>
      <Controller control={control} name="rating" render={({ field }) => <RatingInput label="Rating" required value={field.value} onChange={field.onChange} error={errors.rating?.message} />} />
      <CheckboxField label="Active" {...register('is_active')} />
    </CmsFormFrame>
  )
}

export const testimonialConfig: CmsConfig<Testimonial> = {
  segment: 'testimonials',
  title: 'Testimonials',
  singular: 'Testimonial',
  description: 'Quotes from seafarers and clients shown on the website.',
  layout: 'list',
  label: (r) => r.name,
  summary: (r) => (
    <div className="flex items-center gap-3">
      <Thumb image={r.photo} width={48} height={48} alt="" fit="cover" fallback={r.name.slice(0, 1)} />
      <div className="min-w-0">
        <p className="truncate text-sm font-medium text-slate-900">{r.name}</p>
        <p className="truncate text-xs text-slate-500">{r.position || '—'}</p>
        <RatingDisplay rating={r.rating} />
      </div>
    </div>
  ),
  toPayload: (r, isActive) => ({
    name: r.name,
    position: r.position,
    photo_file_id: r.photo?.file_id ?? null,
    quote: r.quote,
    rating: r.rating,
    is_active: isActive,
  }),
  Form: TestimonialForm,
  emptyHint: 'Add a testimonial to build trust with visitors.',
}

export default function TestimonialsPage() {
  return <CmsListPage config={testimonialConfig} />
}
