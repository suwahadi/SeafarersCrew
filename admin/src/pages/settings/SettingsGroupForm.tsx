import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { errorMessage } from '@/api/client'
import { settingsApi, settingsKey, type ConnectionResult, type SettingGroup, type SettingValues } from '@/api/settings'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Layout'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors } from '@/lib/forms'
import { RecoveryCodesPanel } from './RecoveryCodes'
import { SettingFieldInput } from './SettingFieldInput'
import { buildSettingsSchema, groupLabels, secretStatus, testableGroups, toFormState, toPayload, type FormState } from './settingsForm'

interface SettingsGroupFormProps {
  group: SettingGroup
  onDirtyChange: (dirty: boolean) => void
}

export function SettingsGroupForm({ group, onDirtyChange }: SettingsGroupFormProps) {
  const query = useQuery({ queryKey: settingsKey(group.name), queryFn: ({ signal }) => settingsApi.get(group.name, signal) })

  if (query.isPending) {
    return (
      <div className="flex justify-center p-8">
        <Spinner label="Loading settings" />
      </div>
    )
  }
  if (query.isError) {
    return (
      <p role="alert" className="text-sm text-red-700">
        {errorMessage(query.error)}
      </p>
    )
  }
  return <GroupFields key={query.dataUpdatedAt} group={group} values={query.data} onDirtyChange={onDirtyChange} />
}

function GroupFields({ group, values, onDirtyChange }: { group: SettingGroup; values: SettingValues; onDirtyChange: (dirty: boolean) => void }) {
  const qc = useQueryClient()
  const toast = useToast()
  const schema = useMemo(() => buildSettingsSchema(group), [group])
  const [formError, setFormError] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<ConnectionResult | null>(null)
  const { control, handleSubmit, setError, formState } = useForm<FormState>({
    resolver: zodResolver(schema),
    defaultValues: toFormState(group, values),
  })
  const dirty = formState.isDirty

  useEffect(() => {
    onDirtyChange(dirty)
    return () => onDirtyChange(false)
  }, [dirty, onDirtyChange])

  const save = useMutation({
    mutationFn: (state: FormState) => settingsApi.update(group.name, toPayload(group, state)),
    onSuccess: (saved) => {
      setFormError(null)
      setTestResult(null)
      qc.setQueryData(settingsKey(group.name), saved)
      toast.success(`${groupLabels[group.name] ?? group.name} settings saved.`)
    },
    onError: (err) =>
      setFormError(
        applyServerErrors(
          err,
          setError,
          group.fields.map((f) => f.key),
        ),
      ),
  })

  const test = useMutation({
    mutationFn: () => settingsApi.test(group.name),
    onSuccess: setTestResult,
    onError: (err) => setTestResult({ ok: false, message: errorMessage(err) }),
  })

  return (
    <form onSubmit={handleSubmit((state) => save.mutate(state))} noValidate className="space-y-6">
      {formError ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
          {formError}
        </p>
      ) : null}
      <Card>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          {group.fields.map((field) => (
            <div key={field.key} className={`min-w-0 ${field.kind === 'html' || field.kind === 'textarea' || field.kind === 'list' ? 'md:col-span-2' : ''}`}>
              <Controller
                control={control}
                name={field.key}
                render={({ field: rhf, fieldState }) => (
                  <SettingFieldInput field={field} value={rhf.value} onChange={rhf.onChange} error={fieldState.error?.message} secret={secretStatus(values, field.key)} />
                )}
              />
            </div>
          ))}
        </div>
      </Card>
      {group.name === 'security' ? (
        <Card title="Admin access">
          <div className="space-y-4">
            <p className="text-sm text-slate-700">
              Session idle timeout: <strong>8 hours</strong>
              <span className="ml-2 text-xs text-slate-500">(fixed)</span>
            </p>
            <RecoveryCodesPanel />
          </div>
        </Card>
      ) : null}
      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" loading={save.isPending} disabled={!dirty}>
          Save changes
        </Button>
        {testableGroups.has(group.name) ? (
          <Button variant="secondary" loading={test.isPending} disabled={dirty} onClick={() => test.mutate()}>
            Test connection
          </Button>
        ) : null}
        {dirty ? <span className="text-xs text-orange-700">Unsaved changes. Save before testing the connection.</span> : null}
        {testResult ? (
          <p role="status" className={`text-sm ${testResult.ok ? 'text-green-700' : 'text-red-700'}`}>
            {testResult.ok ? 'Connection successful. ' : 'Connection failed. '}
            {testResult.message}
          </p>
        ) : null}
      </div>
    </form>
  )
}
