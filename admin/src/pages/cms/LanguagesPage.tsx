import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, useForm } from 'react-hook-form'
import { z } from 'zod'
import type { Language } from '@/api/cms'
import { FLAG_ACCEPT } from '@/api/cmsMedia'
import { ImageField } from '@/components/cms/ImageField'
import { Thumb } from '@/components/cms/Thumb'
import { CheckboxField, TextField } from '@/components/ui/Field'
import { Badge } from '@/components/ui/Layout'
import { CmsFormFrame, nullable, useCmsSave } from './cmsForm'
import { CmsListPage, type CmsConfig, type CmsFormProps } from './CmsListPage'

const schema = z
  .object({
    name: z.string().trim().min(1, 'This field is required.').max(80, 'The name may not exceed 80 characters.'),
    code: z.string().trim().regex(/^[a-z]{2,3}(-[A-Za-z]{2,4})?$/, 'Use a Google Translate code such as id, ja or zh-CN.'),
    flag_file_id: z.string(),
    flag_code: z.string().trim().regex(/^([a-z]{2})?$/, 'Use two lowercase letters, such as id or jp.'),
    is_active: z.boolean(),
  })
  .superRefine((v, ctx) => {
    if (v.flag_file_id === '' && v.flag_code === '') ctx.addIssue({ code: 'custom', path: ['flag_code'], message: 'Upload a flag or enter a flag code.' })
  })

type Values = z.infer<typeof schema>

const FIELDS = ['name', 'code', 'flag_file_id', 'flag_code', 'is_active'] as const

export function LanguageForm({ row, onDone }: CmsFormProps<Language>) {
  const isSource = row?.is_source === true
  const { register, control, handleSubmit, setError, formState } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: row?.name ?? '',
      code: row?.code ?? '',
      flag_file_id: row?.flag?.file_id ?? '',
      flag_code: row?.flag_code ?? '',
      is_active: row?.is_active ?? true,
    },
  })
  const { mutation, formError } = useCmsSave<Language, Values>({
    segment: 'languages',
    row,
    fields: FIELDS,
    setError,
    toBody: (v) => ({ name: v.name, code: v.code, flag_file_id: nullable(v.flag_file_id), flag_code: nullable(v.flag_code), is_active: v.is_active }),
    onDone,
    label: 'Language',
  })
  const errors = formState.errors

  return (
    <CmsFormFrame onSubmit={handleSubmit((v) => mutation.mutate(v))} onCancel={onDone} pending={mutation.isPending} formError={formError}>
      {isSource ? (
        <p className="rounded-md bg-slate-50 p-3 text-sm text-slate-700">This is the source language. Its code cannot be changed and it always stays active.</p>
      ) : null}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <TextField label="Name" required maxLength={80} error={errors.name?.message} {...register('name')} />
        <TextField label="Code" required readOnly={isSource} hint="Google Translate code, for example id, ja or zh-CN." error={errors.code?.message} {...register('code')} />
      </div>
      <Controller
        control={control}
        name="flag_file_id"
        render={({ field }) => (
          <ImageField label="Flag (SVG)" context="flag" accept={FLAG_ACCEPT} value={field.value} onChange={field.onChange} image={row?.flag} aspect="4 / 3" error={errors.flag_file_id?.message} />
        )}
      />
      <TextField label="Flag code" maxLength={2} hint="Or enter a two-letter ISO country code to use a bundled flag." error={errors.flag_code?.message} {...register('flag_code')} />
      {isSource ? <p className="text-sm text-slate-600">Always active.</p> : <CheckboxField label="Active" {...register('is_active')} />}
    </CmsFormFrame>
  )
}

const SOURCE_NOTE = 'The source language cannot be deleted, deactivated or recoded.'

export const languageConfig: CmsConfig<Language> = {
  segment: 'languages',
  title: 'Languages',
  singular: 'Language',
  description: 'Languages visitors can pick in the website language switcher.',
  layout: 'list',
  label: (r) => r.name,
  summary: (r) => (
    <div className="flex items-center gap-3">
      <Thumb image={r.flag} width={32} height={24} alt="" fallback={r.flag_code ?? ''} />
      <div className="min-w-0">
        <p className="truncate text-sm font-medium text-slate-900">
          {r.name} {r.is_source ? <Badge tone="blue">Source</Badge> : null}
        </p>
        <p className="text-xs text-slate-500">Code: {r.code}</p>
      </div>
    </div>
  ),
  toPayload: (r, isActive) => ({ name: r.name, code: r.code, flag_file_id: r.flag?.file_id ?? null, flag_code: r.flag_code, is_active: isActive }),
  lockedReason: (r) => (r.is_source ? { active: `${r.name} is the source language and cannot be deactivated.`, remove: `${r.name} is the source language and cannot be deleted.` } : null),
  Form: LanguageForm,
  notice: <p>{SOURCE_NOTE}</p>,
  emptyHint: 'Add a language.',
}

export default function LanguagesPage() {
  return <CmsListPage config={languageConfig} />
}
