import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate } from 'react-router'
import { errorMessage } from '@/api/client'
import { contractsApi, type Contract } from '@/api/contracts'
import { DocumentHistory } from '@/components/documents/DocumentHistory'
import { EmailSendDialog, type EmailAttachment } from '@/components/email/EmailSendDialog'
import { RelatedEmailLog } from '@/components/email/RelatedEmailLog'
import { Button } from '@/components/ui/Button'
import { TextAreaField, TextField } from '@/components/ui/Field'
import { Card } from '@/components/ui/Layout'
import { ConfirmDialog, Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { todayISO } from '@/lib/format'

type Dialog = 'activate' | 'complete' | 'terminate' | 'delete' | 'email' | null

export function ContractActions({ contract }: { contract: Contract }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const [dialog, setDialog] = useState<Dialog>(null)
  const [date, setDate] = useState(todayISO())
  const [reason, setReason] = useState('')
  const [attachments, setAttachments] = useState<EmailAttachment[]>([])
  const [error, setError] = useState<string | null>(null)

  const refresh = (saved: Contract) => {
    qc.setQueryData(['contract', saved.id], saved)
    void qc.invalidateQueries({ queryKey: ['grid', 'contracts'] })
    void qc.invalidateQueries({ queryKey: ['rotation'] })
    setDialog(null)
    setError(null)
  }
  const fail = (err: unknown) => setError(errorMessage(err))

  const activate = useMutation({
    mutationFn: () => contractsApi.activate(contract.id),
    onSuccess: (saved) => {
      refresh(saved)
      toast.success('Contract activated.')
    },
    onError: fail,
  })
  const complete = useMutation({
    mutationFn: () => contractsApi.complete(contract.id, date),
    onSuccess: (saved) => {
      refresh(saved)
      toast.success('Contract completed.')
    },
    onError: fail,
  })
  const terminate = useMutation({
    mutationFn: () => contractsApi.terminate(contract.id, date, reason.trim()),
    onSuccess: (saved) => {
      refresh(saved)
      toast.success('Contract terminated.')
    },
    onError: fail,
  })
  const remove = useMutation({
    mutationFn: () => contractsApi.remove(contract.id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['grid', 'contracts'] })
      qc.removeQueries({ queryKey: ['contract', contract.id] })
      toast.success('Draft contract deleted.')
      void navigate('/contracts', { replace: true })
    },
    onError: fail,
  })

  const open = (d: Dialog) => {
    setError(null)
    setDate(todayISO())
    setReason('')
    setDialog(d)
  }
  const close = () => setDialog(null)
  const reasonMissing = reason.trim() === ''

  return (
    <aside className="space-y-6">
      <Card title="Actions">
        <div className="flex flex-wrap gap-2">
          {contract.status === 'draft' ? <Button onClick={() => open('activate')}>Activate</Button> : null}
          {contract.status === 'active' ? <Button onClick={() => open('complete')}>Complete</Button> : null}
          {contract.status === 'active' ? (
            <Button variant="danger" onClick={() => open('terminate')}>
              Terminate
            </Button>
          ) : null}
          {contract.status === 'draft' ? (
            <Button variant="danger" onClick={() => open('delete')}>
              Delete draft
            </Button>
          ) : null}
          <Button variant="secondary" onClick={() => void navigate(`/documents/new?kind=sea&related_id=${contract.id}`)}>
            Generate SEA
          </Button>
          <Button
            variant="secondary"
            onClick={() => {
              setAttachments([])
              open('email')
            }}
          >
            Send email
          </Button>
        </div>
        {contract.status === 'terminated' && contract.termination_reason ? (
          <p className="mt-3 text-sm text-slate-600">Termination reason: {contract.termination_reason}</p>
        ) : null}
      </Card>
      <Card title="Documents">
        <DocumentHistory
          relatedType="contract"
          relatedId={contract.id}
          onSend={(d) => {
            setAttachments([{ fileId: d.file_id, name: `${d.template_name} (${d.id})` }])
            setDialog('email')
          }}
        />
      </Card>
      <Card title="Emails">
        <RelatedEmailLog relatedType="contract" relatedId={contract.id} />
      </Card>

      <ConfirmDialog
        open={dialog === 'activate'}
        title="Activate contract"
        message={
          <>
            <p>
              Activating locks the contract terms. Afterwards only the planned sign-off date, ports and notes can change, and the crew member becomes On Board.
            </p>
            {error ? <p role="alert" className="mt-2 text-red-700">{error}</p> : null}
          </>
        }
        confirmLabel="Activate"
        loading={activate.isPending}
        onConfirm={() => activate.mutate()}
        onClose={close}
      />
      <ConfirmDialog
        open={dialog === 'delete'}
        title="Delete draft contract"
        message={
          <>
            <p>This permanently deletes draft contract {contract.id}.</p>
            {error ? <p role="alert" className="mt-2 text-red-700">{error}</p> : null}
          </>
        }
        confirmLabel="Delete"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onClose={close}
      />
      <Modal
        open={dialog === 'complete'}
        title="Complete contract"
        onClose={close}
        footer={
          <>
            <Button variant="secondary" onClick={close} disabled={complete.isPending}>
              Cancel
            </Button>
            <Button onClick={() => complete.mutate()} loading={complete.isPending} disabled={date === ''}>
              Complete contract
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <TextField label="Actual sign-off date" type="date" required value={date} onChange={(e) => setDate(e.target.value)} />
          {error ? <p role="alert" className="text-sm text-red-700">{error}</p> : null}
        </div>
      </Modal>
      <Modal
        open={dialog === 'terminate'}
        title="Terminate contract"
        onClose={close}
        footer={
          <>
            <Button variant="secondary" onClick={close} disabled={terminate.isPending}>
              Cancel
            </Button>
            <Button variant="danger" onClick={() => terminate.mutate()} loading={terminate.isPending} disabled={date === '' || reasonMissing}>
              Terminate contract
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <TextField label="Actual sign-off date" type="date" required value={date} onChange={(e) => setDate(e.target.value)} />
          <TextAreaField label="Reason" required value={reason} onChange={(e) => setReason(e.target.value)} maxLength={1000} hint="A reason is required to terminate." />
          {error ? <p role="alert" className="text-sm text-red-700">{error}</p> : null}
        </div>
      </Modal>
      <EmailSendDialog open={dialog === 'email'} onClose={close} relatedType="contract" relatedId={contract.id} attachments={attachments} />
    </aside>
  )
}
