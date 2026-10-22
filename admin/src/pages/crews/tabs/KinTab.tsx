import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { crewKeys, crewsApi, type NextOfKin, type NextOfKinInput } from '@/api/crews'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { CheckboxField, SelectField, TextAreaField, TextField } from '@/components/ui/Field'
import { Badge } from '@/components/ui/Layout'
import { ConfirmDialog, Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors } from '@/lib/forms'
import { titleCase } from '@/lib/format'
import { RowTable } from './RowTable'
import { useRowDelete } from './useRowDelete'

const RELATIONSHIPS = ['spouse', 'parent', 'child', 'sibling', 'other'] as const

const schema = z.object({
  name: z.string().trim().min(1, 'The name is required.').max(150),
  relationship: z.enum(RELATIONSHIPS, 'Choose a relationship.'),
  phone: z
    .string()
    .trim()
    .refine((v) => v === '' || /^\+[1-9][0-9]{6,14}$/.test(v), 'Use E.164 format, e.g. +628123456789.'),
  address: z.string().trim().max(500),
  is_primary: z.boolean(),
})

type Values = z.infer<typeof schema>

const FIELDS = Object.keys(schema.shape) as (keyof Values)[]

function KinForm({ crewId, row, onDone }: { crewId: string; row: NextOfKin | null; onDone: () => void }) {
  const qc = useQueryClient()
  const toast = useToast()
  const [formError, setFormError] = useState<string | null>(null)
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: row ?? { name: '', relationship: 'spouse', phone: '', address: '', is_primary: false },
  })
  const save = useMutation({ mutationFn: (v: NextOfKinInput) => crewsApi.saveNextOfKin(crewId, v, row?.id) })
  const onSubmit = form.handleSubmit(async (v) => {
    setFormError(null)
    try {
      await save.mutateAsync(v)
      void qc.invalidateQueries({ queryKey: crewKeys.tab(crewId, 'kin') })
      toast.success('Next of kin saved.')
      onDone()
    } catch (err) {
      setFormError(applyServerErrors(err, form.setError, FIELDS))
    }
  })
  const e = form.formState.errors
  return (
    <form onSubmit={onSubmit} noValidate className="space-y-3">
      {formError ? <p role="alert" className="rounded-md bg-red-50 p-2 text-sm text-red-700">{formError}</p> : null}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <TextField label="Name" required error={e.name?.message} {...form.register('name')} />
        <SelectField
          label="Relationship"
          required
          options={RELATIONSHIPS.map((r) => ({ value: r, label: titleCase(r) }))}
          error={e.relationship?.message}
          {...form.register('relationship')}
        />
        <TextField label="Phone" type="tel" placeholder="+628123456789" error={e.phone?.message} {...form.register('phone')} />
      </div>
      <TextAreaField label="Address" rows={2} error={e.address?.message} {...form.register('address')} />
      <CheckboxField label="Primary contact" {...form.register('is_primary')} />
      <div className="flex justify-end gap-2 pt-2 max-sm:*:flex-1">
        <Button variant="secondary" onClick={onDone}>
          Cancel
        </Button>
        <Button type="submit" loading={save.isPending}>
          Save
        </Button>
      </div>
    </form>
  )
}

export function KinTab({ crewId }: { crewId: string }) {
  const rows = useQuery({ queryKey: crewKeys.tab(crewId, 'kin'), queryFn: () => crewsApi.nextOfKins(crewId) })
  const [editing, setEditing] = useState<NextOfKin | 'new' | null>(null)
  const del = useRowDelete<NextOfKin>((r) => crewsApi.deleteNextOfKin(crewId, r.id), [crewKeys.tab(crewId, 'kin')], 'Next of kin')

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <Button onClick={() => setEditing('new')}>Add next of kin</Button>
      </div>
      <RowTable<NextOfKin>
        caption="Next of kin"
        loading={rows.isPending}
        rows={rows.data}
        rowKey={(r) => r.id}
        empty="No next of kin yet."
        columns={[
          { header: 'Name', cell: (r) => r.name },
          { header: 'Relationship', cell: (r) => titleCase(r.relationship) },
          { header: 'Phone', cell: (r) => r.phone || '—' },
          { header: 'Address', cell: (r) => r.address || '—', className: 'max-w-xs truncate' },
          { header: 'Primary', cell: (r) => (r.is_primary ? <Badge tone="blue">Primary</Badge> : null) },
        ]}
        actions={(r) => (
          <>
            <Button variant="ghost" size="sm" collapse icon={<Icon name="edit" />} onClick={() => setEditing(r)}>
              Edit
            </Button>
            <Button variant="ghost" size="sm" collapse icon={<Icon name="trash" />} onClick={() => del.ask(r)}>
              Delete
            </Button>
          </>
        )}
      />
      <Modal open={editing !== null} title={editing === 'new' ? 'Add next of kin' : 'Edit next of kin'} onClose={() => setEditing(null)}>
        {editing !== null ? <KinForm crewId={crewId} row={editing === 'new' ? null : editing} onDone={() => setEditing(null)} /> : null}
      </Modal>
      <ConfirmDialog
        open={del.target !== null}
        title="Delete next of kin"
        message={`Delete ${del.target?.name ?? ''}?`}
        confirmLabel="Delete"
        danger
        loading={del.pending}
        onConfirm={del.confirm}
        onClose={del.cancel}
      />
    </div>
  )
}
