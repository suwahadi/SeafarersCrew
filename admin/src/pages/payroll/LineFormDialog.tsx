import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { errorMessage } from '@/api/client'
import { crewKeys, crewsApi } from '@/api/crews'
import { payrollApi, payrollKeys, type PayrollLine, type PayrollLineInput } from '@/api/payroll'
import { CrewPicker } from '@/components/forms/CrewPicker'
import { Button } from '@/components/ui/Button'
import { SelectField, TextField } from '@/components/ui/Field'
import { ConfirmDialog, Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors } from '@/lib/forms'

function amountIssue(value: string, label: string, required: boolean): string | null {
  if (value === '') return required ? `${label} is required.` : null
  const n = Number(value)
  if (!Number.isFinite(n)) return `${label} must be a number.`
  if (n < 0) return `${label} cannot be negative.`
  return null
}

const schema = z
  .object({
    crew: z.object({ id: z.string(), name: z.string() }).nullable(),
    bank_account_id: z.string(),
    amount_usd: z.string().trim(),
    amount_idr: z.string().trim(),
    transfer_date: z.string(),
    remark: z.string().max(300, 'At most 300 characters.'),
  })
  .superRefine((v, ctx) => {
    if (!v.crew)
      ctx.addIssue({
        code: 'custom',
        path: ['crew'],
        message: 'Select the crew member.',
      })
    if (!v.bank_account_id)
      ctx.addIssue({
        code: 'custom',
        path: ['bank_account_id'],
        message: 'Select the bank account.',
      })
    const idr = amountIssue(v.amount_idr, 'IDR', true)
    if (idr) ctx.addIssue({ code: 'custom', path: ['amount_idr'], message: idr })
    const usd = amountIssue(v.amount_usd, 'USD', false)
    if (usd) ctx.addIssue({ code: 'custom', path: ['amount_usd'], message: usd })
  })
type Values = z.infer<typeof schema>
const fields = ['bank_account_id', 'amount_usd', 'amount_idr', 'transfer_date', 'remark'] as const

interface Props {
  periodId: string
  line?: PayrollLine
  onClose: () => void
}

export function LineFormDialog({ periodId, line, onClose }: Props) {
  const qc = useQueryClient()
  const toast = useToast()
  const [formError, setFormError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const { register, control, handleSubmit, setError, setValue, formState } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      crew: line ? { id: line.crew.id, name: line.crew.name } : null,
      bank_account_id: line?.bank_account_id ?? '',
      amount_usd: line?.amount_usd ?? '',
      amount_idr: line?.amount_idr ?? '',
      transfer_date: line?.transfer_date ?? '',
      remark: line?.remark ?? '',
    },
  })
  const { errors } = formState
  const crew = useWatch({ control, name: 'crew' })
  const crewId = crew?.id ?? ''
  const accounts = useQuery({
    queryKey: crewKeys.tab(crewId, 'bank'),
    queryFn: () => crewsApi.bankAccounts(crewId),
    enabled: crewId !== '',
  })

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['grid', 'payroll-lines'] })
    void qc.invalidateQueries({ queryKey: payrollKeys.period(periodId) })
    void qc.invalidateQueries({ queryKey: payrollKeys.facets(periodId) })
  }

  const save = useMutation({
    mutationFn: (input: PayrollLineInput) => (line ? payrollApi.updateLine(line.id, input) : payrollApi.createLine(periodId, input)),
    onSuccess: () => {
      refresh()
      toast.success(line ? 'Row updated.' : 'Row added.')
      onClose()
    },
    onError: (err) => setFormError(applyServerErrors(err, setError, fields)),
  })

  const remove = useMutation({
    mutationFn: () => payrollApi.deleteLine(line?.id ?? ''),
    onSuccess: () => {
      refresh()
      toast.success('Row deleted.')
      onClose()
    },
    onError: (err) => toast.error(errorMessage(err)),
  })

  const submit = handleSubmit((v) => {
    setFormError(null)
    save.mutate({
      crew_id: v.crew?.id ?? '',
      bank_account_id: v.bank_account_id,
      amount_usd: v.amount_usd === '' ? null : v.amount_usd,
      amount_idr: v.amount_idr,
      transfer_date: v.transfer_date === '' ? null : v.transfer_date,
      remark: v.remark.trim(),
    })
  })

  const accountOptions = (accounts.data ?? []).map((a) => ({
    value: a.id,
    label: `${a.bank_name} · ${a.account_number} · ${a.account_name}`,
  }))

  return (
    <>
      <Modal
        open={!confirmDelete}
        size="xl"
        title={line ? `Row ${line.line_no}` : 'Add row'}
        onClose={onClose}
        footer={
          <>
            {line ? (
              <Button variant="danger" onClick={() => setConfirmDelete(true)} disabled={save.isPending} className="mr-auto">
                Delete row
              </Button>
            ) : null}
            <Button variant="secondary" onClick={onClose} disabled={save.isPending}>
              Cancel
            </Button>
            <Button onClick={() => void submit()} loading={save.isPending}>
              Save row
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
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Controller
              control={control}
              name="crew"
              render={({ field }) => (
                <CrewPicker
                  label="Crew member"
                  required
                  value={field.value}
                  onChange={(picked) => {
                    field.onChange(picked)
                    setValue('bank_account_id', '')
                  }}
                  error={errors.crew?.message}
                />
              )}
            />
            <Controller
              control={control}
              name="bank_account_id"
              render={({ field }) => (
                <SelectField
                  label="Bank account"
                  required
                  placeholder={crewId ? 'Select an account' : 'Select the crew member first'}
                  options={accountOptions}
                  disabled={crewId === '' || accounts.isLoading}
                  hint="From the crew profile's Bank tab."
                  error={errors.bank_account_id?.message}
                  name={field.name}
                  ref={field.ref}
                  value={field.value}
                  onBlur={field.onBlur}
                  onChange={(e) => field.onChange(e.target.value)}
                />
              )}
            />
          </div>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
            <TextField label="USD" inputMode="decimal" placeholder="Optional" error={errors.amount_usd?.message} {...register('amount_usd')} />
            <TextField label="IDR" required inputMode="decimal" error={errors.amount_idr?.message} {...register('amount_idr')} />
            <div className="col-span-2 sm:col-span-1">
              <TextField label="Transfer date" type="date" hint="Blank uses the period date." error={errors.transfer_date?.message} {...register('transfer_date')} />
            </div>
          </div>
          <TextField label="Remark" maxLength={300} error={errors.remark?.message} {...register('remark')} />
        </form>
      </Modal>
      <ConfirmDialog
        open={confirmDelete}
        title="Delete this row?"
        message="The transfer row is removed from the period. This cannot be undone."
        confirmLabel="Delete"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onClose={() => setConfirmDelete(false)}
      />
    </>
  )
}
