import { useQuery } from '@tanstack/react-query'
import { useCallback, useMemo, useState } from 'react'
import { errorMessage } from '@/api/client'
import { settingsApi, SETTINGS_SCHEMA_KEY } from '@/api/settings'
import { PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Tabs } from '@/components/ui/Tabs'
import { SettingsGroupForm } from './SettingsGroupForm'
import { groupLabels } from './settingsForm'

export default function SettingsPage() {
  const schema = useQuery({ queryKey: SETTINGS_SCHEMA_KEY, queryFn: ({ signal }) => settingsApi.schema(signal), staleTime: Infinity })
  const [active, setActive] = useState<string | null>(null)
  const [pending, setPending] = useState<string | null>(null)
  const [dirty, setDirty] = useState(false)
  const onDirtyChange = useCallback((value: boolean) => setDirty(value), [])

  const groups = useMemo(() => schema.data ?? [], [schema.data])
  const current = groups.find((g) => g.name === active) ?? groups[0]

  const requestSwitch = (id: string) => {
    if (id === current?.name) return
    if (dirty) setPending(id)
    else setActive(id)
  }

  return (
    <div>
      <PageHeader title="Settings" description="Configure the platform, public website and integrations." />
      {schema.isPending ? (
        <div className="flex justify-center p-8">
          <Spinner label="Loading settings" />
        </div>
      ) : schema.isError ? (
        <p role="alert" className="text-sm text-red-700">
          {errorMessage(schema.error)}
        </p>
      ) : current ? (
        <div className="space-y-6">
          <Tabs
            tabs={groups.map((g) => ({ id: g.name, label: groupLabels[g.name] ?? g.name, dirty: g.name === current.name && dirty }))}
            active={current.name}
            onChange={requestSwitch}
          />
          <SettingsGroupForm key={current.name} group={current} onDirtyChange={onDirtyChange} />
        </div>
      ) : null}
      <ConfirmDialog
        open={pending !== null}
        title="Discard unsaved changes"
        message="This tab has unsaved changes. Switching tabs will discard them."
        confirmLabel="Discard and switch"
        danger
        onClose={() => setPending(null)}
        onConfirm={() => {
          setActive(pending)
          setPending(null)
        }}
      />
    </div>
  )
}
