import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import type { PayrollPeriod, PayrollPeriodInput } from '@/api/payroll'
import { Button } from '@/components/ui/Button'
import { TextField } from '@/components/ui/Field'
import { Modal } from '@/components/ui/Modal'
import { applyServerErrors } from '@/lib/forms'

const schema = z
  .object({
    title: z.string().trim().min(1, 'The title is required.').max(120, 'At most 120 characters.'),
    period_start: z.string().min(1, 'The start date is required.'),
    period_end: z.string().min(1, 'The end date is required.'),
    transfer_date: z.string(),
  })
  .refine((v) => !v.period_start || !v.period_end || v.period_end >= v.period_start, {
    path: ['period_end'],
    message: 'The end date must be on or after the start date.',
  })
type Values = z.infer<typeof schema>
const fields = ['title', 'period_start', 'period_end', 'transfer_date'] as const

interface Props {
  period?: PayrollPeriod
  onSubmit: (input: PayrollPeriodInput) => Promise<unknown>
  onClose: () => void
}

export function PeriodFormDialog({ period, onSubmit, onClose }: Props) {
  const [formError, setFormError] = useState<string | null>(null)
  const { register, handleSubmit, setError, formState } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      title: period?.title ?? '',
      period_start: period?.period_start ?? '',
      period_end: period?.period_end ?? '',
      transfer_date: period?.transfer_date ?? '',
    },
  })
  const { errors, isSubmitting } = formState

  const submit = handleSubmit(async (v) => {
    setFormError(null)
    try {
      await onSubmit({
        title: v.title,
        period_start: v.period_start,
        period_end: v.period_end,
        transfer_date: v.transfer_date || null,
      })
    } catch (err) {
      setFormError(applyServerErrors(err, setError, fields))
    }
  })

  return (
    <Modal
      open
      size="lg"
      title={period ? 'Edit period' : 'New payroll period'}
      onClose={onClose}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button onClick={() => void submit()} loading={isSubmitting}>
            Save period
          </Button>
        </>
      }
    >
      <form onSubmit={(e) => void submit(e)} noValidate className="space-y-4">
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <TextField label="Title" required placeholder="Salary September 2026" error={errors.title?.message} {...register('title')} />
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <TextField label="Period start" type="date" required error={errors.period_start?.message} {...register('period_start')} />
          <TextField label="Period end" type="date" required error={errors.period_end?.message} {...register('period_end')} />
          <div className="col-span-2 sm:col-span-1">
            <TextField label="Transfer date" type="date" error={errors.transfer_date?.message} {...register('transfer_date')} />
          </div>
        </div>
        <p className="text-xs text-slate-500">The transfer date is used for rows that have no date of their own.</p>
      </form>
    </Modal>
  )
}
