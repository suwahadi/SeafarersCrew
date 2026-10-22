import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { crewKeys, crewsApi, type BankAccount, type BankAccountInput } from '@/api/crews'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { CheckboxField, SelectField, TextField } from '@/components/ui/Field'
import { Badge } from '@/components/ui/Layout'
import { ConfirmDialog, Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { useOptions } from '@/hooks/useOptions'
import { applyServerErrors } from '@/lib/forms'
import { RowTable } from './RowTable'
import { useRowDelete } from './useRowDelete'

const schema = z.object({
  bank_name: z.string().trim().min(1, 'The bank name is required.').max(100),
  account_name: z.string().trim().min(1, 'The account name is required.').max(150),
  account_number: z.string().trim().min(1, 'The account number is required.').max(50),
  currency_code: z.string().length(3, 'Choose a currency.'),
  cover_bank: z.string().trim().max(150),
  remarks: z.string().trim().max(300),
  is_primary: z.boolean(),
})

const FIELDS = Object.keys(schema.shape) as (keyof BankAccountInput)[]

const EMPTY: BankAccountInput = { bank_name: '', account_name: '', account_number: '', currency_code: '', cover_bank: '', remarks: '', is_primary: false }

function BankForm({ crewId, row, onDone }: { crewId: string; row: BankAccount | null; onDone: () => void }) {
  const qc = useQueryClient()
  const toast = useToast()
  const { options } = useOptions()
  const [formError, setFormError] = useState<string | null>(null)
  const form = useForm<BankAccountInput>({ resolver: zodResolver(schema), defaultValues: row ?? EMPTY })
  const save = useMutation({ mutationFn: (v: BankAccountInput) => crewsApi.saveBankAccount(crewId, v, row?.id) })
  const onSubmit = form.handleSubmit(async (v) => {
    setFormError(null)
    try {
      await save.mutateAsync(v)
      void qc.invalidateQueries({ queryKey: crewKeys.tab(crewId, 'bank') })
      toast.success('Bank account saved.')
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
        <TextField label="Bank name" required error={e.bank_name?.message} {...form.register('bank_name')} />
        <TextField label="Account name" required error={e.account_name?.message} {...form.register('account_name')} />
        <TextField label="Account number" required error={e.account_number?.message} {...form.register('account_number')} />
        <SelectField
          label="Currency"
          required
          placeholder="Choose…"
          options={options.currencies.map((c) => ({ value: c.code, label: `${c.code} – ${c.name}` }))}
          error={e.currency_code?.message}
          {...form.register('currency_code')}
        />
        <TextField label="Cover bank" error={e.cover_bank?.message} {...form.register('cover_bank')} />
        <TextField label="Remarks" placeholder="e.g. DIRECT" error={e.remarks?.message} {...form.register('remarks')} />
      </div>
      <CheckboxField label="Primary account" hint="Only one account can be primary." {...form.register('is_primary')} />
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

export function BankTab({ crewId }: { crewId: string }) {
  const rows = useQuery({ queryKey: crewKeys.tab(crewId, 'bank'), queryFn: () => crewsApi.bankAccounts(crewId) })
  const [editing, setEditing] = useState<BankAccount | 'new' | null>(null)
  const del = useRowDelete<BankAccount>((r) => crewsApi.deleteBankAccount(crewId, r.id), [crewKeys.tab(crewId, 'bank')], 'Bank account')

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <Button onClick={() => setEditing('new')}>Add bank account</Button>
      </div>
      <RowTable<BankAccount>
        caption="Bank accounts"
        loading={rows.isPending}
        rows={rows.data}
        rowKey={(r) => r.id}
        empty="No bank accounts yet."
        columns={[
          { header: 'Bank', cell: (r) => r.bank_name },
          { header: 'Account name', cell: (r) => r.account_name },
          { header: 'Account number', cell: (r) => <span className="font-mono">{r.account_number}</span> },
          { header: 'Currency', cell: (r) => r.currency_code },
          { header: 'Cover bank', cell: (r) => r.cover_bank || '—' },
          { header: 'Remarks', cell: (r) => r.remarks || '—' },
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
      <Modal open={editing !== null} title={editing === 'new' ? 'Add bank account' : 'Edit bank account'} onClose={() => setEditing(null)}>
        {editing !== null ? <BankForm crewId={crewId} row={editing === 'new' ? null : editing} onDone={() => setEditing(null)} /> : null}
      </Modal>
      <ConfirmDialog
        open={del.target !== null}
        title="Delete bank account"
        message={`Delete the ${del.target?.bank_name ?? ''} account?`}
        confirmLabel="Delete"
        danger
        loading={del.pending}
        onConfirm={del.confirm}
        onClose={del.cancel}
      />
    </div>
  )
}
