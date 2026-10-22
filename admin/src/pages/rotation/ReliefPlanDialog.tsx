import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { Link } from 'react-router'
import { z } from 'zod'
import { ApiError, errorMessage } from '@/api/client'
import { rotationApi, type Contract, type ReliefPlan, type ReliefResult, type ReliefStatus, type ReliefWarning } from '@/api/contracts'
import { CrewPicker } from '@/components/forms/CrewPicker'
import { Button } from '@/components/ui/Button'
import { SelectField, TextAreaField, TextField } from '@/components/ui/Field'
import { Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { formatDate, formatSpan } from '@/lib/format'
import { applyServerErrors } from '@/lib/forms'

const schema = z
  .object({
    reliever: z.object({ id: z.string(), name: z.string() }).nullable(),
    planned_relief_date: z.string().min(1, 'The relief date is required.'),
    joining_port: z.string().max(100),
    status: z.enum(['planned', 'confirmed', 'cancelled']),
    notes: z.string().max(2000),
  })
  .refine((v) => v.reliever !== null, { path: ['reliever'], message: 'Select the reliever.' })
type Values = z.infer<typeof schema>
const fields = ['reliever', 'planned_relief_date', 'joining_port', 'status', 'notes'] as const

export interface ReliefTarget {
  contractId: string
  label: string
  defaultDate: string
}

interface Props {
  target: ReliefTarget
  plan?: ReliefPlan | undefined
  onClose: () => void
}

const REASONS: Record<string, string> = {
  expires_before_contract_end: 'expires before the end of the planned contract',
  expired_on_relief_date: 'is expired on the relief date',
  missing: 'is missing',
}

export function warningText(w: ReliefWarning): string {
  const date = w.expired_date ? ` (${formatDate(w.expired_date)})` : ''
  return `${w.type_name} ${REASONS[w.reason] ?? w.reason}${date}`
}

export function ReliefPlanDialog({ target, plan, onClose }: Props) {
  const qc = useQueryClient()
  const toast = useToast()
  const [formError, setFormError] = useState<string | null>(null)
  const [conflict, setConflict] = useState<string | null>(null)
  const [saved, setSaved] = useState<ReliefResult | null>(null)
  const [newContract, setNewContract] = useState<Contract | null>(null)
  const closed = plan !== undefined && (plan.status === 'completed' || plan.status === 'cancelled')

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      reliever: plan ? { id: plan.reliever.id, name: plan.reliever.name } : null,
      planned_relief_date: plan?.planned_relief_date ?? target.defaultDate,
      joining_port: plan?.joining_port ?? '',
      status: plan && plan.status !== 'completed' ? plan.status : 'planned',
      notes: plan?.notes ?? '',
    },
  })
  const { register, control, handleSubmit, setError, setValue, formState } = form
  const [reliever, date] = useWatch({ control, name: ['reliever', 'planned_relief_date'] })

  const suggestions = useQuery({
    queryKey: ['rotation', 'suggestions', target.contractId, date],
    queryFn: () => rotationApi.suggestions(target.contractId, date),
    enabled: !closed && date !== '',
  })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['rotation'] })
    void qc.invalidateQueries({ queryKey: ['grid', 'relief-plans'] })
  }

  const save = useMutation({
    mutationFn: (v: Values) => {
      const input = {
        contract_id: target.contractId,
        reliever_crew_id: v.reliever?.id ?? '',
        planned_relief_date: v.planned_relief_date,
        joining_port: v.joining_port.trim(),
        status: v.status as ReliefStatus,
        notes: v.notes.trim(),
      }
      return plan ? rotationApi.update(plan.id, input) : rotationApi.create(input)
    },
    onSuccess: (result) => {
      invalidate()
      toast.success('Relief plan saved.')
      if (result.warnings.length > 0) setSaved(result)
      else onClose()
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 409) {
        setConflict(err.message)
        return
      }
      setConflict(null)
      const reliefError = err instanceof ApiError ? err.fields.reliever_crew_id : undefined
      if (reliefError) setError('reliever', { type: 'server', message: reliefError })
      setFormError(applyServerErrors(err, setError, fields))
    },
  })

  const complete = useMutation({
    mutationFn: () => (plan ? rotationApi.complete(plan.id) : Promise.reject(new Error('No relief plan.'))),
    onSuccess: (contract) => {
      invalidate()
      void qc.invalidateQueries({ queryKey: ['grid', 'contracts'] })
      qc.setQueryData(['contract', contract.id], contract)
      setNewContract(contract)
      toast.success('Relief completed. A draft contract was created for the reliever.')
    },
    onError: (err) => setConflict(errorMessage(err)),
  })

  const onSubmit = handleSubmit((values) => {
    setFormError(null)
    setConflict(null)
    save.mutate(values)
  })

  if (newContract) {
    return (
      <Modal open title="Relief completed" onClose={onClose} footer={<Button onClick={onClose}>Close</Button>}>
        <p className="text-sm text-slate-700">
          Draft contract{' '}
          <Link to={`/contracts/${newContract.id}`} className="font-medium text-navy-700 hover:underline">
            {newContract.id}
          </Link>{' '}
          was created for {newContract.crew.name}. Review and activate it when the reliever signs on.
        </p>
      </Modal>
    )
  }

  if (saved) {
    return (
      <Modal open title="Relief plan saved with warnings" onClose={onClose} footer={<Button onClick={onClose}>Close</Button>}>
        <p className="mb-2 text-sm text-slate-700">The plan was saved. Check these items for {saved.reliever.name} before the relief:</p>
        <ul className="list-disc space-y-1 pl-5 text-sm text-orange-800">
          {saved.warnings.map((w) => (
            <li key={`${w.kind}-${w.type_name}`}>{warningText(w)}</li>
          ))}
        </ul>
      </Modal>
    )
  }

  const candidates = suggestions.data?.candidates ?? []
  const canComplete = plan !== undefined && (plan.status === 'planned' || plan.status === 'confirmed')

  return (
    <Modal
      open
      size="lg"
      title={plan ? 'Relief plan' : 'Plan relief'}
      onClose={onClose}
      footer={
        <>
          {canComplete ? (
            <Button variant="secondary" onClick={() => complete.mutate()} loading={complete.isPending} className="mr-auto">
              Complete relief
            </Button>
          ) : null}
          <Button variant="secondary" onClick={onClose} disabled={save.isPending}>
            {closed ? 'Close' : 'Cancel'}
          </Button>
          {closed ? null : (
            <Button onClick={() => void onSubmit()} loading={save.isPending}>
              Save relief plan
            </Button>
          )}
        </>
      }
    >
      <form onSubmit={onSubmit} noValidate className="space-y-4">
        <p className="text-sm text-slate-600">Relieving {target.label}.</p>
        {closed ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">This relief plan is {plan.status} and cannot be changed.</p> : null}
        {conflict ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {conflict}
          </p>
        ) : null}
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <fieldset disabled={closed} className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <TextField label="Relief date" type="date" required error={formState.errors.planned_relief_date?.message} {...register('planned_relief_date')} />
            <TextField label="Joining port" error={formState.errors.joining_port?.message} {...register('joining_port')} />
          </div>
          <Controller
            control={control}
            name="reliever"
            render={({ field, fieldState }) => (
              <CrewPicker label="Reliever" required value={field.value} onChange={field.onChange} error={fieldState.error?.message} disabled={closed} />
            )}
          />
          {closed ? null : (
            <section aria-label="Suggested relievers">
              <h3 className="mb-1 text-sm font-medium text-slate-700">Suggested relievers (available crew in the same rank)</h3>
              {suggestions.isError ? <p className="text-sm text-red-700">{errorMessage(suggestions.error)}</p> : null}
              {suggestions.isSuccess && candidates.length === 0 ? <p className="text-sm text-slate-500">No available crew match this rank.</p> : null}
              <ul className="max-h-48 divide-y divide-slate-100 overflow-y-auto rounded-md ring-1 ring-slate-200">
                {candidates.map((c) => (
                  <li key={c.id}>
                    <button
                      type="button"
                      aria-pressed={reliever?.id === c.id}
                      onClick={() => setValue('reliever', { id: c.id, name: c.full_name }, { shouldValidate: true })}
                      className={`flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm hover:bg-slate-50 ${reliever?.id === c.id ? 'bg-navy-50' : ''}`}
                    >
                      <span>
                        <span className="font-medium">{c.full_name}</span> <span className="text-slate-500">{c.id}</span>
                        <span className="block text-xs text-slate-500">
                          {c.nationality}
                          {c.age !== null ? ` · ${c.age} y` : ''}
                        </span>
                      </span>
                      <span className="text-xs text-slate-600">Sea service in rank: {formatSpan(c.sea_service_in_rank)}</span>
                    </button>
                  </li>
                ))}
              </ul>
              <p className="mt-1 text-xs text-slate-500">Certificate and document compliance is checked when the plan is saved.</p>
            </section>
          )}
          <SelectField
            label="Status"
            options={[
              { value: 'planned', label: 'Planned' },
              { value: 'confirmed', label: 'Confirmed' },
              { value: 'cancelled', label: 'Cancelled' },
            ]}
            error={formState.errors.status?.message}
            {...register('status')}
          />
          <TextAreaField label="Notes" error={formState.errors.notes?.message} {...register('notes')} />
        </fieldset>
      </form>
    </Modal>
  )
}
