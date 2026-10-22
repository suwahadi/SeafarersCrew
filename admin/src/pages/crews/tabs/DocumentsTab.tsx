import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { z } from 'zod'
import { errorMessage } from '@/api/client'
import { crewKeys, crewsApi, type CrewDocument, type CrewDocumentInput, type DocumentKind } from '@/api/crews'
import { openFile } from '@/api/files'
import { FileField } from '@/components/forms/FileField'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { SelectField, TextField } from '@/components/ui/Field'
import { ExpiryBadge } from '@/components/ui/Layout'
import { ConfirmDialog, Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { useOptions } from '@/hooks/useOptions'
import { formatDate, titleCase } from '@/lib/format'
import { applyServerErrors } from '@/lib/forms'
import { RowTable } from './RowTable'
import { useRowDelete } from './useRowDelete'

const isoDate = /^\d{4}-\d{2}-\d{2}$/
const optionalDate = z.string().refine((v) => v === '' || isoDate.test(v), 'Use a valid date.')

const schema = z
  .object({
    type_id: z.string().min(1, 'Choose a type.'),
    number: z.string().trim().min(1, 'The number is required.').max(60),
    issued_date: optionalDate,
    expired_date: optionalDate,
    issuing_authority: z.string().trim().max(150),
    place_of_issue: z.string().trim().max(100),
    file_id: z.string(),
  })
  .refine((v) => v.issued_date === '' || v.expired_date === '' || v.expired_date >= v.issued_date, {
    path: ['expired_date'],
    message: 'The expiry date must be on or after the issued date.',
  })

const EMPTY: CrewDocumentInput = { type_id: '', number: '', issued_date: '', expired_date: '', issuing_authority: '', place_of_issue: '', file_id: '' }

const FIELDS = Object.keys(EMPTY) as (keyof CrewDocumentInput)[]

const LABELS: Record<DocumentKind, { singular: string; plural: string; key: string }> = {
  documents: { singular: 'document', plural: 'Documents', key: 'documents' },
  certificates: { singular: 'certificate', plural: 'Certificates', key: 'certificates' },
}

function toValues(row: CrewDocument | null): CrewDocumentInput {
  if (!row) return EMPTY
  return {
    type_id: row.type_id,
    number: row.number,
    issued_date: row.issued_date ?? '',
    expired_date: row.expired_date ?? '',
    issuing_authority: row.issuing_authority ?? '',
    place_of_issue: row.place_of_issue,
    file_id: row.file_id ?? '',
  }
}

interface FormProps {
  crewId: string
  kind: DocumentKind
  row: CrewDocument | null
  onDone: () => void
}

function DocumentForm({ crewId, kind, row, onDone }: FormProps) {
  const qc = useQueryClient()
  const toast = useToast()
  const { options } = useOptions()
  const [formError, setFormError] = useState<string | null>(null)
  const form = useForm<CrewDocumentInput>({ resolver: zodResolver(schema), defaultValues: toValues(row) })
  const save = useMutation({ mutationFn: (v: CrewDocumentInput) => crewsApi.saveDocument(crewId, kind, v, row?.id) })
  const types = kind === 'documents' ? options.document_types : options.certificate_types
  const label = LABELS[kind]

  const onSubmit = form.handleSubmit(async (v) => {
    setFormError(null)
    try {
      await save.mutateAsync(v)
      void qc.invalidateQueries({ queryKey: crewKeys.tab(crewId, label.key) })
      void qc.invalidateQueries({ queryKey: crewKeys.tab(crewId, 'compliance') })
      void qc.invalidateQueries({ queryKey: ['grid', 'crews'] })
      toast.success(`${titleCase(label.singular)} saved.`)
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
        <SelectField
          label="Type"
          required
          placeholder="Choose…"
          options={types.map((t) => ({ value: t.id, label: t.name }))}
          error={e.type_id?.message}
          {...form.register('type_id')}
        />
        <TextField label="Number" required error={e.number?.message} {...form.register('number')} />
        <TextField label="Issued date" type="date" error={e.issued_date?.message} {...form.register('issued_date')} />
        <TextField label="Expiry date" type="date" error={e.expired_date?.message} {...form.register('expired_date')} />
        <TextField label="Issuing authority" error={e.issuing_authority?.message} {...form.register('issuing_authority')} />
        <TextField label="Place of issue" error={e.place_of_issue?.message} {...form.register('place_of_issue')} />
      </div>
      <Controller
        control={form.control}
        name="file_id"
        render={({ field, fieldState }) => (
          <FileField
            label="Scanned copy"
            context="crew_document"
            fileId={field.value}
            onChange={(id) => field.onChange(id)}
            hint="PDF or image."
            error={fieldState.error?.message}
          />
        )}
      />
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

export function DocumentExpiry({ row }: { row: Pick<CrewDocument, 'has_expiry' | 'expired_date' | 'badge' | 'days_left'> }) {
  if (!row.has_expiry && !row.expired_date) return <span className="text-slate-500">No expiry</span>
  if (!row.expired_date) return <span className="text-slate-500">—</span>
  return (
    <span className="inline-flex items-center gap-2">
      {formatDate(row.expired_date)}
      <ExpiryBadge badge={row.badge} daysLeft={row.days_left} />
    </span>
  )
}

export function DocumentsTab({ crewId, kind }: { crewId: string; kind: DocumentKind }) {
  const label = LABELS[kind]
  const toast = useToast()
  const rows = useQuery({ queryKey: crewKeys.tab(crewId, label.key), queryFn: () => crewsApi.documents(crewId, kind) })
  const [editing, setEditing] = useState<CrewDocument | 'new' | null>(null)
  const del = useRowDelete<CrewDocument>(
    (r) => crewsApi.deleteDocument(crewId, kind, r.id),
    [crewKeys.tab(crewId, label.key), crewKeys.tab(crewId, 'compliance'), ['grid', 'crews']],
    titleCase(label.singular),
  )
  const view = useMutation({ mutationFn: openFile, onError: (err) => toast.error(errorMessage(err)) })

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <Button onClick={() => setEditing('new')}>Add {label.singular}</Button>
      </div>
      <RowTable<CrewDocument>
        caption={label.plural}
        loading={rows.isPending}
        rows={rows.data}
        rowKey={(r) => r.id}
        empty={`No ${label.plural.toLowerCase()} yet.`}
        columns={[
          { header: 'Type', cell: (r) => r.type_name },
          { header: 'Number', cell: (r) => <span className="font-mono">{r.number}</span> },
          { header: 'Issued', cell: (r) => formatDate(r.issued_date) },
          { header: 'Expiry', cell: (r) => <DocumentExpiry row={r} /> },
          { header: 'Place of issue', cell: (r) => r.place_of_issue || '—' },
          { header: 'File', cell: (r) => (r.file_id ? 'Attached' : '—') },
        ]}
        actions={(r) => (
          <>
            {r.file_id ? (
              <Button variant="ghost" size="sm" collapse icon={<Icon name="eye" />} onClick={() => view.mutate(r.file_id ?? '')}>
                View
              </Button>
            ) : null}
            <Button variant="ghost" size="sm" collapse icon={<Icon name="edit" />} onClick={() => setEditing(r)}>
              Edit
            </Button>
            <Button variant="ghost" size="sm" collapse icon={<Icon name="trash" />} onClick={() => del.ask(r)}>
              Delete
            </Button>
          </>
        )}
      />
      <Modal open={editing !== null} title={editing === 'new' ? `Add ${label.singular}` : `Edit ${label.singular}`} size="lg" onClose={() => setEditing(null)}>
        {editing !== null ? <DocumentForm crewId={crewId} kind={kind} row={editing === 'new' ? null : editing} onDone={() => setEditing(null)} /> : null}
      </Modal>
      <ConfirmDialog
        open={del.target !== null}
        title={`Delete ${label.singular}`}
        message={`Delete ${del.target?.type_name ?? ''} ${del.target?.number ?? ''}?`}
        confirmLabel="Delete"
        danger
        loading={del.pending}
        onConfirm={del.confirm}
        onClose={del.cancel}
      />
    </div>
  )
}
