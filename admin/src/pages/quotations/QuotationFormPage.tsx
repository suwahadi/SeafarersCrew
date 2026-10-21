import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useFieldArray, useForm, useWatch } from 'react-hook-form'
import { Link, useNavigate, useParams } from 'react-router'
import { ApiError, errorMessage } from '@/api/client'
import { quotationsApi, type Quotation } from '@/api/quotations'
import { Button } from '@/components/ui/Button'
import { SelectField, TextAreaField, TextField } from '@/components/ui/Field'
import { Card, PageHeader } from '@/components/ui/Layout'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { useOptions } from '@/hooks/useOptions'
import { formatMoney, todayISO } from '@/lib/format'
import { applyServerErrors } from '@/lib/forms'
import { addDays } from '@/lib/timeline'
import { quotationTotals } from '@/lib/quotationMath'
import { StatusBadge } from '../contracts/ContractStatusBadge'
import { QuotationActions } from './QuotationActions'
import { emptyDailyLine, emptyLumpLine, emptyQuotation, quotationSchema, quotationToValues, valuesToInput, type QuotationValues } from './quotationForm'

const formFields = ['client_company_id', 'vessel_id', 'currency_code', 'quotation_date', 'valid_until', 'tax_percent', 'discount_amount', 'terms', 'notes', 'lines'] as const

export default function QuotationFormPage() {
  const { id } = useParams()
  const quotation = useQuery({ queryKey: ['quotation', id], queryFn: () => quotationsApi.get(id ?? ''), enabled: id !== undefined })
  if (id !== undefined && quotation.isPending) {
    return (
      <div className="flex justify-center p-12 text-navy-800">
        <Spinner size="lg" label="Loading quotation" />
      </div>
    )
  }
  if (quotation.isError) return <p role="alert" className="text-sm text-red-700">{errorMessage(quotation.error)}</p>
  return <QuotationForm key={quotation.data?.updated_at ?? 'new'} quotation={quotation.data} />
}

function QuotationForm({ quotation }: { quotation: Quotation | undefined }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const { options } = useOptions()
  const [formError, setFormError] = useState<string | null>(null)
  const locked = quotation !== undefined && quotation.status !== 'draft'

  const today = todayISO()
  const form = useForm<QuotationValues>({
    resolver: zodResolver(quotationSchema),
    defaultValues: quotation ? quotationToValues(quotation) : emptyQuotation(today, addDays(today, 30)),
  })
  const { register, control, handleSubmit, setError, formState } = form
  const lines = useFieldArray({ control, name: 'lines' })
  const [currency, rows, tax, discount] = useWatch({ control, name: ['currency_code', 'lines', 'tax_percent', 'discount_amount'] })
  const decimals = options.currencies.find((c) => c.code === currency)?.decimals ?? 2
  const totals = quotationTotals(rows, tax, discount, decimals)
  const companies = options.companies

  const save = useMutation({
    mutationFn: (values: QuotationValues) => {
      const input = valuesToInput(values)
      return quotation ? quotationsApi.update(quotation.id, input) : quotationsApi.create(input)
    },
    onSuccess: (saved) => {
      toast.success(quotation ? 'Quotation saved.' : `Quotation ${saved.id} created.`)
      qc.setQueryData(['quotation', saved.id], saved)
      void qc.invalidateQueries({ queryKey: ['grid', 'quotations'] })
      if (!quotation) void navigate(`/quotations/${saved.id}`, { replace: true })
    },
    onError: (err) => {
      if (err instanceof ApiError) {
        for (const [key, message] of Object.entries(err.fields)) {
          if (key.startsWith('lines.')) setError(key as `lines.${number}.amount`, { type: 'server', message })
        }
      }
      setFormError(applyServerErrors(err, setError, formFields))
    },
  })

  const onSubmit = handleSubmit((values) => {
    setFormError(null)
    save.mutate(values)
  })

  return (
    <>
      <PageHeader
        title={quotation ? `Quotation ${quotation.id}` : 'New quotation'}
        description={quotation ? `${quotation.client_name}${quotation.vessel_name ? ` · ${quotation.vessel_name}` : ''}` : 'Price crew supply for a client.'}
        actions={
          <>
            {quotation ? <StatusBadge status={quotation.status} /> : null}
            <Link to="/quotations" className="text-sm text-navy-700 hover:underline">
              Back to quotations
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
          {locked ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">Only draft quotations can be edited.</p> : null}
          <fieldset disabled={locked} className="min-w-0 space-y-6">
            <Card title="Details">
              <div className="grid gap-4 md:grid-cols-2">
                <SelectField
                  label="Client"
                  required
                  placeholder="Select a client"
                  options={companies.map((c) => ({ value: c.id, label: c.name }))}
                  error={formState.errors.client_company_id?.message}
                  {...register('client_company_id')}
                />
                <SelectField
                  label="Vessel"
                  placeholder="No vessel"
                  options={options.vessels.map((v) => ({ value: v.id, label: v.name }))}
                  error={formState.errors.vessel_id?.message}
                  {...register('vessel_id')}
                />
                <SelectField
                  label="Currency"
                  required
                  options={options.currencies.map((c) => ({ value: c.code, label: `${c.code} – ${c.name}` }))}
                  error={formState.errors.currency_code?.message}
                  {...register('currency_code')}
                />
                <div className="grid grid-cols-2 gap-4">
                  <TextField label="Quotation date" type="date" required error={formState.errors.quotation_date?.message} {...register('quotation_date')} />
                  <TextField label="Valid until" type="date" required error={formState.errors.valid_until?.message} {...register('valid_until')} />
                </div>
              </div>
            </Card>
            <Card title="Lines">
              <div className="space-y-3">
                {lines.fields.map((f, i) => {
                  const row = rows[i]
                  const lineErrors = formState.errors.lines?.[i]
                  return (
                    <div key={f.id} className="rounded-md bg-slate-50 p-3 ring-1 ring-slate-200">
                      <div className="mb-2 flex items-center justify-between">
                        <span className="text-xs font-semibold text-slate-600 uppercase">{row?.line_type === 'lump_sum' ? 'Lump sum' : 'Daily rate'}</span>
                        <Button variant="ghost" size="sm" onClick={() => lines.remove(i)} aria-label={`Remove line ${i + 1}`} disabled={locked}>
                          Remove
                        </Button>
                      </div>
                      {row?.line_type === 'lump_sum' ? (
                        <div className="grid gap-3 md:grid-cols-[1fr_10rem]">
                          <TextField label={`Line ${i + 1} description`} error={lineErrors?.description?.message} {...register(`lines.${i}.description`)} />
                          <TextField label={`Line ${i + 1} amount`} inputMode="decimal" error={lineErrors?.amount?.message} {...register(`lines.${i}.amount`)} />
                        </div>
                      ) : (
                        <div className="grid gap-3 md:grid-cols-4">
                          <SelectField
                            label={`Line ${i + 1} rank`}
                            placeholder="Select a rank"
                            options={options.ranks.map((r) => ({ value: r.id, label: r.name }))}
                            error={lineErrors?.rank_id?.message}
                            {...register(`lines.${i}.rank_id`)}
                          />
                          <TextField label={`Line ${i + 1} crew count`} type="number" min={1} error={lineErrors?.crew_count?.message} {...register(`lines.${i}.crew_count`)} />
                          <TextField label={`Line ${i + 1} daily rate`} inputMode="decimal" error={lineErrors?.daily_rate?.message} {...register(`lines.${i}.daily_rate`)} />
                          <TextField label={`Line ${i + 1} days`} type="number" min={1} error={lineErrors?.days?.message} {...register(`lines.${i}.days`)} />
                        </div>
                      )}
                      <p className="mt-2 text-right text-sm tabular-nums text-slate-700">
                        Line total (preview): {formatMoney(totals.lines[i] ?? '0', currency, decimals)}
                      </p>
                    </div>
                  )
                })}
              </div>
              {formState.errors.lines?.message ? <p role="alert" className="mt-2 text-xs text-red-600">{formState.errors.lines.message}</p> : null}
              {formState.errors.lines?.root?.message ? <p role="alert" className="mt-2 text-xs text-red-600">{formState.errors.lines.root.message}</p> : null}
              <div className="mt-3 flex gap-2">
                <Button variant="secondary" size="sm" onClick={() => lines.append({ ...emptyDailyLine })} disabled={locked}>
                  Add daily rate line
                </Button>
                <Button variant="secondary" size="sm" onClick={() => lines.append({ ...emptyLumpLine })} disabled={locked}>
                  Add lump sum line
                </Button>
              </div>
            </Card>
            <Card title="Totals">
              <div className="grid gap-4 md:grid-cols-2">
                <TextField label="Discount" inputMode="decimal" error={formState.errors.discount_amount?.message} {...register('discount_amount')} />
                <TextField label="Tax (%)" inputMode="decimal" error={formState.errors.tax_percent?.message} {...register('tax_percent')} />
              </div>
              <dl className="mt-4 space-y-1 text-sm tabular-nums">
                <TotalRow label="Subtotal" value={formatMoney(totals.subtotal, currency, decimals)} />
                <TotalRow label="Discount" value={formatMoney(`-${totals.discount}`, currency, decimals)} />
                <TotalRow label="Tax on subtotal less discount" value={formatMoney(totals.tax, currency, decimals)} />
                <TotalRow label="Grand total (preview)" value={formatMoney(totals.grandTotal, currency, decimals)} strong />
              </dl>
              {totals.discountExceedsSubtotal ? <p role="alert" className="mt-2 text-xs text-orange-700">The discount is larger than the subtotal.</p> : null}
              <p className="mt-2 text-xs text-slate-500">Preview only: the server recalculates every amount when the quotation is saved.</p>
            </Card>
            <Card title="Terms and notes">
              <div className="space-y-4">
                <TextAreaField label="Terms" rows={6} error={formState.errors.terms?.message} {...register('terms')} />
                <TextAreaField label="Internal notes" error={formState.errors.notes?.message} {...register('notes')} />
              </div>
            </Card>
          </fieldset>
          {locked ? null : (
            <div className="flex justify-end">
              <Button type="submit" loading={save.isPending}>
                {quotation ? 'Save quotation' : 'Create draft quotation'}
              </Button>
            </div>
          )}
        </form>
        {quotation ? <QuotationActions quotation={quotation} /> : null}
      </div>
    </>
  )
}

function TotalRow({ label, value, strong = false }: { label: string; value: string; strong?: boolean }) {
  return (
    <div className={`flex justify-between ${strong ? 'border-t border-slate-200 pt-1 font-semibold' : ''}`}>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  )
}
