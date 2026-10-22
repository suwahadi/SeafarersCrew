import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Controller, useFieldArray, useForm, useWatch } from 'react-hook-form'
import { Link, useNavigate, useParams } from 'react-router'
import { contractsApi, type Contract } from '@/api/contracts'
import { ApiError, errorMessage } from '@/api/client'
import { CrewPicker } from '@/components/forms/CrewPicker'
import { Button } from '@/components/ui/Button'
import { inputClass, SelectField, TextAreaField, TextField } from '@/components/ui/Field'
import { Card, PageHeader } from '@/components/ui/Layout'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { useOptions } from '@/hooks/useOptions'
import { addMonths, isFieldEditable, monthlyAllowance, totalMonthlyPay, type ContractField } from '@/lib/contractRules'
import { formatMoney } from '@/lib/format'
import { applyServerErrors } from '@/lib/forms'
import { ContractActions } from './ContractActions'
import { contractSchema, contractToValues, emptyContract, valuesToInput, type ContractValues } from './contractForm'
import { StatusBadge } from './ContractStatusBadge'

const formFields = [
  'crew',
  'vessel_id',
  'rank_id',
  'principal_id',
  'employer_company_id',
  'sign_on_date',
  'duration_months',
  'planned_sign_off_date',
  'point_of_hire',
  'sign_on_port',
  'sign_off_port',
  'currency_code',
  'basic_salary',
  'working_hours_week',
  'leave_days_month',
  'probation_months',
  'notes',
  'allowances',
] as const

export default function ContractFormPage() {
  const { id } = useParams()
  const contract = useQuery({ queryKey: ['contract', id], queryFn: () => contractsApi.get(id ?? ''), enabled: id !== undefined })
  if (id !== undefined && contract.isPending) {
    return (
      <div className="flex justify-center p-12 text-navy-800">
        <Spinner size="lg" label="Loading contract" />
      </div>
    )
  }
  if (contract.isError) return <p role="alert" className="text-sm text-red-700">{errorMessage(contract.error)}</p>
  return <ContractForm key={contract.data?.updated_at ?? 'new'} contract={contract.data} />
}

function ContractForm({ contract }: { contract: Contract | undefined }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const { options } = useOptions()
  const [formError, setFormError] = useState<string | null>(null)
  const [plannedTouched, setPlannedTouched] = useState(contract !== undefined)
  const status = contract?.status
  const editable = (field: ContractField) => isFieldEditable(status, field)
  const readOnly = status === 'completed' || status === 'terminated'

  const form = useForm<ContractValues>({
    resolver: zodResolver(contractSchema),
    defaultValues: contract ? contractToValues(contract) : emptyContract,
  })
  const { register, control, handleSubmit, setValue, setError, formState } = form
  const allowances = useFieldArray({ control, name: 'allowances' })
  const [signOn, duration, currency, basic, rows] = useWatch({ control, name: ['sign_on_date', 'duration_months', 'currency_code', 'basic_salary', 'allowances'] })

  const currencies = options.currencies
  const decimals = currencies.find((c) => c.code === currency)?.decimals ?? 2
  const allowanceTypes = options.allowance_types
  const basisOf = (typeId: string) => allowanceTypes.find((t) => t.id === typeId)?.basis ?? 'fixed_month'
  const companies = options.companies

  const syncPlanned = (on: string, months: string) => {
    if (plannedTouched) return
    const planned = addMonths(on, Number(months))
    if (planned) setValue('planned_sign_off_date', planned)
  }

  const save = useMutation({
    mutationFn: (values: ContractValues) => {
      const input = valuesToInput(values, status, contract)
      return contract ? contractsApi.update(contract.id, input) : contractsApi.create(input)
    },
    onSuccess: (saved) => {
      toast.success(contract ? 'Contract saved.' : `Contract ${saved.id} created.`)
      qc.setQueryData(['contract', saved.id], saved)
      void qc.invalidateQueries({ queryKey: ['grid', 'contracts'] })
      if (!contract) void navigate(`/contracts/${saved.id}`, { replace: true })
    },
    onError: (err) => {
      if (err instanceof ApiError && err.code === 'contract_locked') {
        setFormError(`${err.message} Locked fields: ${Object.keys(err.fields).join(', ') || 'see above'}.`)
        return
      }
      setFormError(applyServerErrors(err, setError, formFields))
      const crewError = err instanceof ApiError ? err.fields.crew_id : undefined
      if (crewError) setError('crew', { type: 'server', message: crewError })
    },
  })

  const onSubmit = handleSubmit((values) => {
    setFormError(null)
    save.mutate(values)
  })

  const companyOptions = (types: string[]) => companies.filter((c) => types.includes(c.type)).map((c) => ({ value: c.id, label: c.name }))

  return (
    <>
      <PageHeader
        title={contract ? `Contract ${contract.id}` : 'New contract'}
        description={contract ? `${contract.crew.name} · ${contract.rank.name} on ${contract.vessel.name}` : 'Draft contracts stay editable until sign-on is confirmed.'}
        actions={
          <>
            {contract ? <StatusBadge status={contract.status} /> : null}
            <Link to="/contracts" className="text-sm text-navy-700 hover:underline">
              Back to contracts
            </Link>
          </>
        }
      />
      <div className="grid gap-6 xl:grid-cols-[1fr_22rem]">
        <form onSubmit={onSubmit} noValidate className="space-y-6">
          {formError ? (
            <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
              {formError}
            </p>
          ) : null}
          {status === 'active' ? (
            <p className="rounded-md bg-blue-50 p-3 text-sm text-blue-800">
              This contract is active: only the planned sign-off date, ports and notes can change.
            </p>
          ) : null}
          {readOnly ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">This contract is closed and read-only.</p> : null}
          <Card title="Assignment">
            <div className="grid gap-4 md:grid-cols-2">
              <Controller
                control={control}
                name="crew"
                render={({ field, fieldState }) => (
                  <CrewPicker label="Crew member" required value={field.value} onChange={field.onChange} error={fieldState.error?.message} disabled={!editable('crew_id')} />
                )}
              />
              <SelectField
                label="Vessel"
                required
                disabled={!editable('vessel_id')}
                placeholder="Select a vessel"
                options={options.vessels.map((v) => ({ value: v.id, label: `${v.name} (IMO ${v.imo_number})` }))}
                error={formState.errors.vessel_id?.message}
                {...register('vessel_id', {
                  onChange: (e: { target: { value: string } }) => {
                    const v = options.vessels.find((x) => x.id === e.target.value)
                    if (v?.principal_id && form.getValues('principal_id') === '') setValue('principal_id', v.principal_id)
                  },
                })}
              />
              <SelectField
                label="Rank"
                required
                disabled={!editable('rank_id')}
                placeholder="Select a rank"
                options={options.ranks.map((r) => ({ value: r.id, label: r.name }))}
                error={formState.errors.rank_id?.message}
                {...register('rank_id')}
              />
              <SelectField
                label="Principal"
                required
                disabled={!editable('principal_id')}
                placeholder="Select a principal"
                options={companyOptions(['principal', 'both'])}
                error={formState.errors.principal_id?.message}
                {...register('principal_id')}
              />
              <SelectField
                label="Employer"
                required
                disabled={!editable('employer_company_id')}
                placeholder="Select the employer"
                options={companyOptions(['company', 'both'])}
                error={formState.errors.employer_company_id?.message}
                {...register('employer_company_id')}
              />
              <TextField label="Point of hire" disabled={!editable('point_of_hire')} error={formState.errors.point_of_hire?.message} {...register('point_of_hire')} />
            </div>
          </Card>
          <Card title="Period">
            <div className="grid gap-4 md:grid-cols-3">
              <TextField
                label="Sign-on date"
                type="date"
                required
                disabled={!editable('sign_on_date')}
                error={formState.errors.sign_on_date?.message}
                {...register('sign_on_date', { onChange: (e: { target: { value: string } }) => syncPlanned(e.target.value, duration) })}
              />
              <TextField
                label="Duration (months)"
                type="number"
                min={1}
                max={60}
                required
                disabled={!editable('duration_months')}
                error={formState.errors.duration_months?.message}
                {...register('duration_months', { onChange: (e: { target: { value: string } }) => syncPlanned(signOn, e.target.value) })}
              />
              <TextField
                label="Planned sign-off"
                type="date"
                disabled={!editable('planned_sign_off_date')}
                hint={plannedTouched ? 'Edited manually.' : 'Sign-on plus duration; you can change it.'}
                error={formState.errors.planned_sign_off_date?.message}
                {...register('planned_sign_off_date', { onChange: () => setPlannedTouched(true) })}
              />
              <TextField label="Place of sign-on" disabled={!editable('sign_on_port')} error={formState.errors.sign_on_port?.message} {...register('sign_on_port')} />
              <TextField label="Place of sign-off" disabled={!editable('sign_off_port')} error={formState.errors.sign_off_port?.message} {...register('sign_off_port')} />
              <TextField label="Probation (months)" type="number" min={0} max={24} disabled={!editable('probation_months')} error={formState.errors.probation_months?.message} {...register('probation_months')} />
              <TextField label="Working hours per week" type="number" min={0} max={168} disabled={!editable('working_hours_week')} error={formState.errors.working_hours_week?.message} {...register('working_hours_week')} />
              <TextField label="Leave days per month" inputMode="decimal" disabled={!editable('leave_days_month')} error={formState.errors.leave_days_month?.message} {...register('leave_days_month')} />
            </div>
          </Card>
          <Card title="Wages">
            <div className="grid gap-4 md:grid-cols-3">
              <SelectField
                label="Currency"
                required
                disabled={!editable('currency_code')}
                options={currencies.map((c) => ({ value: c.code, label: `${c.code} – ${c.name}` }))}
                error={formState.errors.currency_code?.message}
                {...register('currency_code')}
              />
              <TextField label="Basic salary (monthly)" inputMode="decimal" required disabled={!editable('basic_salary')} error={formState.errors.basic_salary?.message} {...register('basic_salary')} />
            </div>
            <table className="mt-4 min-w-full text-sm">
              <caption className="sr-only">Allowances</caption>
              <thead>
                <tr className="text-left text-xs font-semibold tracking-wide text-slate-500 uppercase">
                  <th className="py-1 pr-2">Allowance</th>
                  <th className="py-1 pr-2">Basis</th>
                  <th className="py-1 pr-2">Value</th>
                  <th className="py-1 pr-2 text-right">Monthly (preview)</th>
                  <th className="py-1" />
                </tr>
              </thead>
              <tbody>
                {allowances.fields.map((f, i) => {
                  const row = rows[i]
                  const basis = basisOf(row?.allowance_type_id ?? '')
                  return (
                    <tr key={f.id} className="align-top">
                      <td className="py-1 pr-2">
                        <select
                          aria-label={`Allowance ${i + 1} type`}
                          disabled={!editable('allowances')}
                          className={inputClass}
                          {...register(`allowances.${i}.allowance_type_id`)}
                        >
                          <option value="">Select…</option>
                          {allowanceTypes.map((t) => (
                            <option key={t.id} value={t.id}>
                              {t.name}
                            </option>
                          ))}
                        </select>
                        {formState.errors.allowances?.[i]?.allowance_type_id ? (
                          <p className="text-xs text-red-600">{formState.errors.allowances[i].allowance_type_id.message}</p>
                        ) : null}
                      </td>
                      <td className="py-1 pr-2 leading-8 text-slate-600">{basis === 'fixed_day' ? 'Per day × 30' : basis === 'percent_basic' ? '% of basic' : 'Per month'}</td>
                      <td className="py-1 pr-2">
                        <input
                          aria-label={`Allowance ${i + 1} value`}
                          inputMode="decimal"
                          disabled={!editable('allowances')}
                          className={`${inputClass} w-36!`}
                          {...register(`allowances.${i}.value`)}
                        />
                        {formState.errors.allowances?.[i]?.value ? <p className="text-xs text-red-600">{formState.errors.allowances[i].value.message}</p> : null}
                      </td>
                      <td className="py-1 pr-2 text-right leading-8 tabular-nums">{formatMoney(monthlyAllowance(basis, row?.value ?? '', basic, decimals), currency, decimals)}</td>
                      <td className="py-1 text-right">
                        {editable('allowances') ? (
                          <Button variant="ghost" size="sm" onClick={() => allowances.remove(i)} aria-label={`Remove allowance ${i + 1}`}>
                            Remove
                          </Button>
                        ) : null}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
              <tfoot>
                <tr className="border-t border-slate-200 font-semibold">
                  <td colSpan={3} className="py-2">
                    Total monthly pay (preview, {currency})
                  </td>
                  <td className="py-2 text-right tabular-nums">
                    {formatMoney(totalMonthlyPay(basic, rows.map((r) => ({ basis: basisOf(r.allowance_type_id), value: r.value })), decimals), currency, decimals)}
                  </td>
                  <td />
                </tr>
              </tfoot>
            </table>
            {editable('allowances') ? (
              <Button variant="secondary" size="sm" className="mt-2" onClick={() => allowances.append({ allowance_type_id: '', value: '' })}>
                Add allowance
              </Button>
            ) : null}
            <p className="mt-2 text-xs text-slate-500">Amounts are recalculated by the server when saved; no currency conversion is applied.</p>
          </Card>
          <Card title="Notes">
            <TextAreaField label="Notes" disabled={!editable('notes')} error={formState.errors.notes?.message} {...register('notes')} />
          </Card>
          {readOnly ? null : (
            <div className="flex justify-end gap-2">
              <Button type="submit" loading={save.isPending}>
                {contract ? 'Save contract' : 'Create draft contract'}
              </Button>
            </div>
          )}
        </form>
        {contract ? <ContractActions contract={contract} /> : null}
      </div>
    </>
  )
}
