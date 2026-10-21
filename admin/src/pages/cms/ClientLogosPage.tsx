import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { z } from 'zod'
import type { ClientLogo } from '@/api/cms'
import { LOGO_ACCEPT } from '@/api/cmsMedia'
import { ImageField } from '@/components/cms/ImageField'
import { Thumb } from '@/components/cms/Thumb'
import { CheckboxField, TextField } from '@/components/ui/Field'
import { isHttpsUrl } from '@/lib/links'
import { CmsFormFrame, useCmsSave } from './cmsForm'
import { CmsListPage, type CmsConfig, type CmsFormProps } from './CmsListPage'

const schema = z.object({
  name: z.string().trim().min(1, 'This field is required.').max(120, 'The name may not exceed 120 characters.'),
  logo_file_id: z.string().min(1, 'Upload the logo.'),
  url: z.string().trim().refine((v) => v === '' || isHttpsUrl(v), 'Enter a valid https:// address.'),
  is_active: z.boolean(),
})

type Values = z.infer<typeof schema>

const FIELDS = ['name', 'logo_file_id', 'url', 'is_active'] as const

export function ClientLogoForm({ row, onDone }: CmsFormProps<ClientLogo>) {
  const { register, control, handleSubmit, setError, formState } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { name: row?.name ?? '', logo_file_id: row?.logo?.file_id ?? '', url: row?.url ?? '', is_active: row?.is_active ?? true },
  })
  const { mutation, formError } = useCmsSave<ClientLogo, Values>({
    segment: 'client-logos',
    row,
    fields: FIELDS,
    setError,
    toBody: (v) => ({ name: v.name, logo_file_id: v.logo_file_id, url: v.url, is_active: v.is_active }),
    onDone,
    label: 'Logo',
  })
  const errors = formState.errors

  return (
    <CmsFormFrame onSubmit={handleSubmit((v) => mutation.mutate(v))} onCancel={onDone} pending={mutation.isPending} formError={formError}>
      <TextField label="Name" required maxLength={120} error={errors.name?.message} {...register('name')} />
      <Controller
        control={control}
        name="logo_file_id"
        render={({ field }) => (
          <ImageField label="Logo" required context="logo" accept={LOGO_ACCEPT} value={field.value} onChange={field.onChange} image={row?.logo} aspect="3 / 2" hint="PNG, JPG, WebP or SVG." error={errors.logo_file_id?.message} />
        )}
      />
      <TextField label="Website" type="url" hint="Optional. Must start with https://" error={errors.url?.message} {...register('url')} />
      <CheckboxField label="Active" {...register('is_active')} />
    </CmsFormFrame>
  )
}

export const clientLogoConfig: CmsConfig<ClientLogo> = {
  segment: 'client-logos',
  title: 'Client logos',
  singular: 'Logo',
  description: 'Logos of clients and principals shown on the website.',
  layout: 'grid',
  label: (r) => r.name,
  summary: (r) => (
    <div className="flex items-center gap-3">
      <Thumb image={r.logo} width={96} height={64} alt={r.name} />
      <div className="min-w-0">
        <p className="truncate text-sm font-medium text-slate-900">{r.name}</p>
        {r.url ? <p className="truncate text-xs text-slate-500">{r.url}</p> : null}
      </div>
    </div>
  ),
  toPayload: (r, isActive) => ({ name: r.name, logo_file_id: r.logo?.file_id ?? '', url: r.url, is_active: isActive }),
  Form: ClientLogoForm,
  emptyHint: 'Add a client logo.',
}

export default function ClientLogosPage() {
  return <CmsListPage config={clientLogoConfig} />
}
