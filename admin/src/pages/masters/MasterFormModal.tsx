import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { masterApi, optionsQuery, OPTIONS_KEY, type MasterRecord } from '@/api/masters'
import { FileUploadField } from '@/components/forms/FileUploadField'
import { Button } from '@/components/ui/Button'
import { CheckboxField, SelectField, TextAreaField, TextField } from '@/components/ui/Field'
import { Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors } from '@/lib/forms'
import type { FieldDef, MasterConfig } from './config'
import { buildSchema, defaultValues, toPayload, type FormValues } from './masterForm'

interface MasterFormModalProps {
  config: MasterConfig
  row: MasterRecord | null
  open: boolean
  onClose: () => void
}

export function MasterFormModal({ config, row, open, onClose }: MasterFormModalProps) {
  return (
    <Modal open={open} title={`${row ? 'Edit' : 'New'} ${config.singular.toLowerCase()}`} onClose={onClose} size="lg">
      {open ? <MasterForm key={row?.id ?? 'new'} config={config} row={row} onDone={onClose} /> : null}
    </Modal>
  )
}

export function MasterForm({ config, row, onDone }: { config: MasterConfig; row: MasterRecord | null; onDone: () => void }) {
  const qc = useQueryClient()
  const toast = useToast()
  const needsOptions = config.fields.some((f) => f.type === 'reference')
  const options = useQuery({ ...optionsQuery, enabled: needsOptions })
  const schema = useMemo(() => buildSchema(config.fields), [config.fields])
  const [formError, setFormError] = useState<string | null>(null)
  const { register, control, handleSubmit, setError, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: defaultValues(config.fields, row ?? undefined),
  })

  const save = useMutation({
    mutationFn: (values: FormValues) => {
      const body = toPayload(config.fields, values, row ?? undefined)
      const svc = masterApi(config.segment)
      return row ? svc.update(row.id, body) : svc.create(body)
    },
    onSuccess: async () => {
      await Promise.all([qc.invalidateQueries({ queryKey: ['grid', config.segment] }), qc.invalidateQueries({ queryKey: OPTIONS_KEY })])
      toast.success(`${config.singular} saved.`)
      onDone()
    },
    onError: (err) => setFormError(applyServerErrors(err, setError, config.fields.map((f) => f.name))),
  })

  const renderField = (f: FieldDef) => {
    const error = formState.errors[f.name]?.message
    const disabled = f.createOnly === true && row !== null
    switch (f.type) {
      case 'checkbox':
        return <CheckboxField key={f.name} label={f.label} hint={f.hint} {...register(f.name)} />
      case 'textarea':
        return <TextAreaField key={f.name} label={f.label} hint={f.hint} error={error} required={f.required} {...register(f.name)} />
      case 'select':
        return <SelectField key={f.name} label={f.label} options={f.options ?? []} error={error} required={f.required} {...register(f.name)} />
      case 'reference':
        return (
          <SelectField
            key={f.name}
            label={f.label}
            options={options.data && f.source ? f.source(options.data) : []}
            placeholder="— None —"
            error={error}
            {...register(f.name)}
          />
        )
      case 'logo':
        return (
          <Controller
            key={f.name}
            control={control}
            name={f.name}
            render={({ field }) => (
              <FileUploadField label={f.label} context="logo" hint={f.hint} error={error} value={typeof field.value === 'string' ? field.value : ''} onChange={field.onChange} />
            )}
          />
        )
      case 'color':
        return (
          <div key={f.name} className="flex items-end gap-2">
            <Controller
              control={control}
              name={f.name}
              render={({ field }) => (
                <input
                  type="color"
                  aria-label={`${f.label} picker`}
                  value={typeof field.value === 'string' && /^#[0-9A-Fa-f]{6}$/.test(field.value) ? field.value : '#64748B'}
                  onChange={(e) => field.onChange(e.target.value.toUpperCase())}
                  className="h-9 w-12 rounded border border-slate-300"
                />
              )}
            />
            <div className="flex-1">
              <TextField label={f.label} error={error} required={f.required} {...register(f.name)} />
            </div>
          </div>
        )
      default:
        return (
          <TextField
            key={f.name}
            label={f.label}
            hint={f.hint}
            error={error}
            required={f.required}
            disabled={disabled}
            inputMode={f.type === 'number' || f.type === 'imo' || f.type === 'decimal' ? 'numeric' : undefined}
            type={f.type === 'email' ? 'email' : f.type === 'url' ? 'url' : 'text'}
            maxLength={f.maxLength}
            {...register(f.name)}
          />
        )
    }
  }

  return (
    <form onSubmit={handleSubmit((v) => save.mutate(v))} noValidate className="space-y-4">
      {formError ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
          {formError}
        </p>
      ) : null}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">{config.fields.map(renderField)}</div>
      <div className="flex justify-end gap-2 border-t border-slate-200 pt-4">
        <Button variant="secondary" onClick={onDone} disabled={save.isPending}>
          Cancel
        </Button>
        <Button type="submit" loading={save.isPending}>
          Save
        </Button>
      </div>
    </form>
  )
}
