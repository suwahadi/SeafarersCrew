import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import type { Stat } from '@/api/cms'
import { CheckboxField, TextField } from '@/components/ui/Field'
import { formatStat } from '@/lib/stats'
import { CmsFormFrame, useCmsSave } from './cmsForm'
import { CmsListPage, type CmsConfig, type CmsFormProps } from './CmsListPage'

const schema = z.object({
  number: z
    .string()
    .trim()
    .regex(/^\d+$/, 'Enter a whole number.')
    .refine((v) => Number(v) <= 1_000_000_000, 'The number may not exceed 1,000,000,000.'),
  suffix: z.string().trim().max(10, 'The suffix may not exceed 10 characters.'),
  label: z.string().trim().min(1, 'This field is required.').max(80, 'The label may not exceed 80 characters.'),
  is_active: z.boolean(),
})

type Values = z.infer<typeof schema>

const FIELDS = ['number', 'suffix', 'label', 'is_active'] as const

export function StatForm({ row, onDone }: CmsFormProps<Stat>) {
  const { register, control, handleSubmit, setError, formState } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { number: row ? String(row.number) : '', suffix: row?.suffix ?? '', label: row?.label ?? '', is_active: row?.is_active ?? true },
  })
  const { mutation, formError } = useCmsSave<Stat, Values>({
    segment: 'stats',
    row,
    fields: FIELDS,
    setError,
    toBody: (v) => ({ number: Number(v.number), suffix: v.suffix, label: v.label, is_active: v.is_active }),
    onDone,
    label: 'Stat',
  })
  const [number, suffix, label] = useWatch({ control, name: ['number', 'suffix', 'label'] })
  const errors = formState.errors

  return (
    <CmsFormFrame onSubmit={handleSubmit((v) => mutation.mutate(v))} onCancel={onDone} pending={mutation.isPending} formError={formError}>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <TextField label="Number" required inputMode="numeric" error={errors.number?.message} {...register('number')} />
        <TextField label="Suffix" maxLength={10} hint="For example +, % or k." error={errors.suffix?.message} {...register('suffix')} />
      </div>
      <TextField label="Label" required maxLength={80} error={errors.label?.message} {...register('label')} />
      <p className="text-sm text-slate-600">
        Preview: <span className="font-semibold text-slate-900">{formatStat({ number: /^\d+$/.test(number) ? Number(number) : 0, suffix, label })}</span>
      </p>
      <CheckboxField label="Active" {...register('is_active')} />
    </CmsFormFrame>
  )
}

export const statConfig: CmsConfig<Stat> = {
  segment: 'stats',
  title: 'Stats',
  singular: 'Stat',
  description: 'Headline numbers shown on the website.',
  layout: 'list',
  label: (r) => r.label,
  summary: (r) => <p className="text-sm font-medium text-slate-900">{formatStat(r)}</p>,
  toPayload: (r, isActive) => ({ number: r.number, suffix: r.suffix, label: r.label, is_active: isActive }),
  Form: StatForm,
  emptyHint: 'Add a headline number such as 1,200+ Seafarers Deployed.',
}

export default function StatsPage() {
  return <CmsListPage config={statConfig} />
}
