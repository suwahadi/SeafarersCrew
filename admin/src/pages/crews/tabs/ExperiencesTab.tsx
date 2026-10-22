import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { crewKeys, crewsApi, type Experience, type ExperienceInput } from '@/api/crews'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { SelectField, TextAreaField, TextField } from '@/components/ui/Field'
import { Badge, Card } from '@/components/ui/Layout'
import { ConfirmDialog, Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { useOptions } from '@/hooks/useOptions'
import { formatDate, formatSpan } from '@/lib/format'
import { applyServerErrors } from '@/lib/forms'
import { RowTable } from './RowTable'
import { useRowDelete } from './useRowDelete'

const isoDate = /^\d{4}-\d{2}-\d{2}$/

const schema = z
  .object({
    vessel_name: z.string().trim().min(1, 'The vessel name is required.').max(150),
    vessel_id: z.string(),
    rank_id: z.string().min(1, 'Choose a rank.'),
    vessel_type_id: z.string(),
    grt_hp: z.string().trim().max(60),
    company_id: z.string(),
    company_name: z.string().trim().max(150),
    principal_id: z.string(),
    principal_name: z.string().trim().max(150),
    sign_on_date: z.string().refine((v) => isoDate.test(v), 'The sign-on date is required.'),
    sign_off_date: z.string().refine((v) => v === '' || isoDate.test(v), 'Use a valid date.'),
    remarks: z.string().trim().max(500),
  })
  .refine((v) => v.sign_off_date === '' || v.sign_off_date >= v.sign_on_date, {
    path: ['sign_off_date'],
    message: 'The sign-off date must be on or after the sign-on date.',
  })

const EMPTY: ExperienceInput = {
  vessel_name: '',
  vessel_id: '',
  rank_id: '',
  vessel_type_id: '',
  grt_hp: '',
  company_id: '',
  company_name: '',
  principal_id: '',
  principal_name: '',
  sign_on_date: '',
  sign_off_date: '',
  remarks: '',
}

const FIELDS = Object.keys(EMPTY) as (keyof ExperienceInput)[]

function toValues(row: Experience | null): ExperienceInput {
  if (!row) return EMPTY
  return {
    vessel_name: row.vessel_name,
    vessel_id: row.vessel_id ?? '',
    rank_id: row.rank_id,
    vessel_type_id: row.vessel_type_id ?? '',
    grt_hp: row.grt_hp,
    company_id: row.company_id ?? '',
    company_name: row.company_name,
    principal_id: row.principal_id ?? '',
    principal_name: row.principal_name,
    sign_on_date: row.sign_on_date,
    sign_off_date: row.sign_off_date ?? '',
    remarks: row.remarks,
  }
}

function invalidateAll(qc: ReturnType<typeof useQueryClient>, crewId: string) {
  void qc.invalidateQueries({ queryKey: crewKeys.tab(crewId, 'experiences') })
  void qc.invalidateQueries({ queryKey: ['grid', 'crews'] })
}

function ExperienceForm({ crewId, row, onDone }: { crewId: string; row: Experience | null; onDone: () => void }) {
  const qc = useQueryClient()
  const toast = useToast()
  const { options } = useOptions()
  const [formError, setFormError] = useState<string | null>(null)
  const form = useForm<ExperienceInput>({ resolver: zodResolver(schema), defaultValues: toValues(row) })
  const save = useMutation({ mutationFn: (v: ExperienceInput) => crewsApi.saveExperience(crewId, v, row?.id) })
  const onSubmit = form.handleSubmit(async (v) => {
    setFormError(null)
    try {
      await save.mutateAsync(v)
      invalidateAll(qc, crewId)
      toast.success('Experience saved.')
      onDone()
    } catch (err) {
      setFormError(applyServerErrors(err, form.setError, FIELDS))
    }
  })
  const e = form.formState.errors
  return (
    <form onSubmit={onSubmit} noValidate className="space-y-3">
      {formError ? (
        <p role="alert" className="rounded-md bg-red-50 p-2 text-sm text-red-700">
          {formError}
        </p>
      ) : null}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <TextField label="Vessel name" required error={e.vessel_name?.message} {...form.register('vessel_name')} />
        <SelectField
          label="Rank"
          required
          placeholder="Choose…"
          options={options.ranks.map((r) => ({ value: r.id, label: r.name }))}
          error={e.rank_id?.message}
          {...form.register('rank_id')}
        />
        <SelectField
          label="Vessel type"
          placeholder="Not set"
          options={options.vessel_types.map((t) => ({ value: t.id, label: t.name }))}
          error={e.vessel_type_id?.message}
          {...form.register('vessel_type_id')}
        />
        <TextField label="GRT / HP" error={e.grt_hp?.message} {...form.register('grt_hp')} />
        <TextField label="Company" error={e.company_name?.message} {...form.register('company_name')} />
        <TextField label="Principal" error={e.principal_name?.message} {...form.register('principal_name')} />
        <TextField label="Sign-on date" type="date" required error={e.sign_on_date?.message} {...form.register('sign_on_date')} />
        <TextField label="Sign-off date" type="date" error={e.sign_off_date?.message} {...form.register('sign_off_date')} />
      </div>
      <TextAreaField label="Remarks" rows={2} error={e.remarks?.message} {...form.register('remarks')} />
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

export function ExperiencesTab({ crewId }: { crewId: string }) {
  const data = useQuery({ queryKey: crewKeys.tab(crewId, 'experiences'), queryFn: () => crewsApi.seaService(crewId) })
  const [editing, setEditing] = useState<Experience | 'new' | null>(null)
  const del = useRowDelete<Experience>(
    (r) => crewsApi.deleteExperience(crewId, r.id),
    [crewKeys.tab(crewId, 'experiences'), ['grid', 'crews']],
    'Experience',
  )

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm text-slate-600">
          Total sea service: <strong>{formatSpan(data.data?.total)}</strong>
        </p>
        <Button onClick={() => setEditing('new')}>Add experience</Button>
      </div>
      <RowTable<Experience>
        caption="Experiences"
        loading={data.isPending}
        rows={data.data?.rows}
        rowKey={(r) => r.id}
        empty="No sea service recorded yet."
        columns={[
          { header: 'Vessel', cell: (r) => r.vessel_name },
          { header: 'Rank', cell: (r) => r.rank_name },
          { header: 'Type', cell: (r) => r.vessel_type || '—' },
          { header: 'Company', cell: (r) => r.company_name || '—' },
          { header: 'Principal', cell: (r) => r.principal_name || '—' },
          { header: 'Sign on', cell: (r) => formatDate(r.sign_on_date) },
          { header: 'Sign off', cell: (r) => (r.sign_off_date ? formatDate(r.sign_off_date) : <Badge tone="blue">On board</Badge>) },
          { header: 'Duration', cell: (r) => formatSpan(r.duration) },
          { header: 'Source', cell: (r) => (r.source === 'contract' ? <Badge tone="neutral">Contract</Badge> : <Badge tone="neutral">Manual</Badge>) },
        ]}
        actions={(r) =>
          r.source === 'manual' ? (
            <>
              <Button variant="ghost" size="sm" collapse icon={<Icon name="edit" />} onClick={() => setEditing(r)}>
                Edit
              </Button>
              <Button variant="ghost" size="sm" collapse icon={<Icon name="trash" />} onClick={() => del.ask(r)}>
                Delete
              </Button>
            </>
          ) : (
            <span className="text-xs text-slate-500">Read-only</span>
          )
        }
      />
      <Card title="Sea service by rank">
        {data.data && data.data.by_rank.length > 0 ? (
          <table className="min-w-full text-sm">
            <caption className="sr-only">Sea service by rank</caption>
            <thead>
              <tr className="text-left text-xs tracking-wide text-slate-500 uppercase">
                <th scope="col" className="py-1 pr-4">
                  Rank
                </th>
                <th scope="col" className="py-1">
                  Total
                </th>
              </tr>
            </thead>
            <tbody>
              {data.data.by_rank.map((r) => (
                <tr key={r.rank_id}>
                  <td className="py-1 pr-4">{r.rank_name}</td>
                  <td className="py-1">{formatSpan(r.total)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="text-sm text-slate-500">No totals yet.</p>
        )}
      </Card>
      <Modal open={editing !== null} title={editing === 'new' ? 'Add experience' : 'Edit experience'} size="lg" onClose={() => setEditing(null)}>
        {editing !== null ? <ExperienceForm crewId={crewId} row={editing === 'new' ? null : editing} onDone={() => setEditing(null)} /> : null}
      </Modal>
      <ConfirmDialog
        open={del.target !== null}
        title="Delete experience"
        message={`Delete the ${del.target?.vessel_name ?? ''} entry?`}
        confirmLabel="Delete"
        danger
        loading={del.pending}
        onConfirm={del.confirm}
        onClose={del.cancel}
      />
    </div>
  )
}
