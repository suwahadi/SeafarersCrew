import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { Link } from 'react-router'
import { z } from 'zod'
import type { HeroSlide } from '@/api/cms'
import { ImageField } from '@/components/cms/ImageField'
import { Thumb } from '@/components/cms/Thumb'
import { CheckboxField, TextAreaField, TextField } from '@/components/ui/Field'
import { buttonPairError, isValidLink, LINK_FORMAT_MESSAGE } from '@/lib/links'
import { CmsFormFrame, nullable, useCmsSave } from './cmsForm'
import { CmsListPage, type CmsConfig, type CmsFormProps } from './CmsListPage'

const schema = z
  .object({
    image_file_id: z.string().min(1, 'Upload the desktop image.'),
    mobile_image_file_id: z.string(),
    heading: z.string().trim().max(160, 'The heading may not exceed 160 characters.'),
    subheading: z.string().trim().max(300, 'The subheading may not exceed 300 characters.'),
    button_label: z.string().trim().max(60, 'The button label may not exceed 60 characters.'),
    button_link: z.string().trim(),
    is_active: z.boolean(),
  })
  .superRefine((v, ctx) => {
    const pair = buttonPairError(v.button_label, v.button_link)
    if (pair) ctx.addIssue({ code: 'custom', path: [pair.field], message: pair.message })
    else if (v.button_link !== '' && !isValidLink(v.button_link)) ctx.addIssue({ code: 'custom', path: ['button_link'], message: LINK_FORMAT_MESSAGE })
  })

type Values = z.infer<typeof schema>

const FIELDS = ['image_file_id', 'mobile_image_file_id', 'heading', 'subheading', 'button_label', 'button_link', 'is_active'] as const

function body(v: Values): Record<string, unknown> {
  return {
    image_file_id: v.image_file_id,
    mobile_image_file_id: nullable(v.mobile_image_file_id),
    heading: v.heading,
    subheading: v.subheading,
    button_label: v.button_label,
    button_link: v.button_link,
    is_active: v.is_active,
  }
}

export function HeroSlideForm({ row, onDone }: CmsFormProps<HeroSlide>) {
  const { register, control, handleSubmit, setError, formState } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      image_file_id: row?.image?.file_id ?? '',
      mobile_image_file_id: row?.mobile_image?.file_id ?? '',
      heading: row?.heading ?? '',
      subheading: row?.subheading ?? '',
      button_label: row?.button_label ?? '',
      button_link: row?.button_link ?? '',
      is_active: row?.is_active ?? true,
    },
  })
  const { mutation, formError } = useCmsSave<HeroSlide, Values>({ segment: 'hero-slides', row, fields: FIELDS, setError, toBody: body, onDone, label: 'Slide' })
  const errors = formState.errors

  return (
    <CmsFormFrame onSubmit={handleSubmit((v) => mutation.mutate(v))} onCancel={onDone} pending={mutation.isPending} formError={formError}>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Controller
          control={control}
          name="image_file_id"
          render={({ field }) => (
            <ImageField label="Desktop image" required value={field.value} onChange={field.onChange} image={row?.image} aspect="16 / 9" hint="Landscape 16:9, at least 1600 px wide." error={errors.image_file_id?.message} />
          )}
        />
        <Controller
          control={control}
          name="mobile_image_file_id"
          render={({ field }) => (
            <ImageField label="Mobile image" value={field.value} onChange={field.onChange} image={row?.mobile_image} aspect="4 / 5" hint="Optional portrait image, 9:16 or 4:5." error={errors.mobile_image_file_id?.message} />
          )}
        />
      </div>
      <TextField label="Heading" maxLength={160} error={errors.heading?.message} {...register('heading')} />
      <TextAreaField label="Subheading" maxLength={300} error={errors.subheading?.message} {...register('subheading')} />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <TextField label="Button label" maxLength={60} error={errors.button_label?.message} {...register('button_label')} />
        <TextField label="Button link" hint="/path, https://…, mailto: or tel:" error={errors.button_link?.message} {...register('button_link')} />
      </div>
      <CheckboxField label="Active" {...register('is_active')} />
    </CmsFormFrame>
  )
}

export const heroSlideConfig: CmsConfig<HeroSlide> = {
  segment: 'hero-slides',
  title: 'Hero slides',
  singular: 'Slide',
  description: 'Slides shown at the top of the home page, in this order.',
  layout: 'list',
  compact: true,
  label: (r) => r.heading || 'Untitled slide',
  summary: (r) => <Thumb image={r.image} width={96} height={54} alt={r.heading} fit="cover" />,
  toPayload: (r, isActive) => ({
    image_file_id: r.image?.file_id ?? '',
    mobile_image_file_id: r.mobile_image?.file_id ?? null,
    heading: r.heading,
    subheading: r.subheading,
    button_label: r.button_label,
    button_link: r.button_link,
    is_active: isActive,
  }),
  Form: HeroSlideForm,
  notice: (
    <p>
      The autoplay interval is set under Website in{' '}
      <Link to="/settings" className="font-medium text-navy-700 underline">
        Settings
      </Link>
      .
    </p>
  ),
  emptyHint: 'Add a slide to show a hero image on the home page.',
}

export default function HeroSlidesPage() {
  return <CmsListPage config={heroSlideConfig} />
}
