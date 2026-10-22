import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate } from 'react-router'
import { errorMessage } from '@/api/client'
import { QUOTATION_TRANSITIONS, quotationsApi, type Quotation, type QuotationStatus } from '@/api/quotations'
import { DocumentHistory } from '@/components/documents/DocumentHistory'
import { EmailSendDialog, type EmailAttachment } from '@/components/email/EmailSendDialog'
import { RelatedEmailLog } from '@/components/email/RelatedEmailLog'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { Card } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'

const LABELS: Record<QuotationStatus, string> = { draft: 'Draft', sent: 'Mark as sent', accepted: 'Mark as accepted', rejected: 'Mark as rejected', expired: 'Mark as expired' }

type Dialog = 'status' | 'delete' | 'email' | null

export function QuotationActions({ quotation }: { quotation: Quotation }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const [dialog, setDialog] = useState<Dialog>(null)
  const [target, setTarget] = useState<QuotationStatus | null>(null)
  const [attachments, setAttachments] = useState<EmailAttachment[]>([])

  const setStatus = useMutation({
    mutationFn: (status: QuotationStatus) => quotationsApi.setStatus(quotation.id, status),
    onSuccess: (saved) => {
      qc.setQueryData(['quotation', saved.id], saved)
      void qc.invalidateQueries({ queryKey: ['grid', 'quotations'] })
      toast.success(`Quotation marked as ${saved.status}.`)
      setDialog(null)
    },
    onError: (err) => {
      toast.error(errorMessage(err))
      setDialog(null)
    },
  })
  const remove = useMutation({
    mutationFn: () => quotationsApi.remove(quotation.id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['grid', 'quotations'] })
      qc.removeQueries({ queryKey: ['quotation', quotation.id] })
      toast.success('Draft quotation deleted.')
      void navigate('/quotations', { replace: true })
    },
    onError: (err) => {
      toast.error(errorMessage(err))
      setDialog(null)
    },
  })

  return (
    <aside className="space-y-6">
      <Card title="Actions">
        <div className="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap">
          {QUOTATION_TRANSITIONS[quotation.status].map((s) => (
            <Button
              key={s}
              icon={<Icon name={s === 'rejected' || s === 'expired' ? 'ban' : 'check'} />}
              variant={s === 'rejected' || s === 'expired' ? 'secondary' : 'primary'}
              onClick={() => {
                setTarget(s)
                setDialog('status')
              }}
            >
              {LABELS[s]}
            </Button>
          ))}
          <Button variant="secondary" icon={<Icon name="file" />} onClick={() => void navigate(`/documents/new?kind=quotation&related_id=${quotation.id}`)}>
            Generate PDF
          </Button>
          <Button
            variant="secondary"
            icon={<Icon name="mail" />}
            onClick={() => {
              setAttachments([])
              setDialog('email')
            }}
          >
            Send email
          </Button>
          {quotation.status === 'draft' ? (
            <Button variant="danger" icon={<Icon name="trash" />} onClick={() => setDialog('delete')}>
              Delete draft
            </Button>
          ) : null}
        </div>
      </Card>
      <Card title="Documents">
        <DocumentHistory
          relatedType="quotation"
          relatedId={quotation.id}
          onSend={(d) => {
            setAttachments([{ fileId: d.file_id, name: `${d.template_name} (${d.id})` }])
            setDialog('email')
          }}
        />
      </Card>
      <Card title="Emails">
        <RelatedEmailLog relatedType="quotation" relatedId={quotation.id} />
      </Card>
      <ConfirmDialog
        open={dialog === 'status'}
        title="Change quotation status"
        message={target ? `Mark quotation ${quotation.id} as ${target}? A ${target} quotation cannot be edited.` : ''}
        confirmLabel="Confirm"
        loading={setStatus.isPending}
        onConfirm={() => target && setStatus.mutate(target)}
        onClose={() => setDialog(null)}
      />
      <ConfirmDialog
        open={dialog === 'delete'}
        title="Delete draft quotation"
        message={`This permanently deletes draft quotation ${quotation.id}.`}
        confirmLabel="Delete"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onClose={() => setDialog(null)}
      />
      <EmailSendDialog open={dialog === 'email'} onClose={() => setDialog(null)} relatedType="quotation" relatedId={quotation.id} attachments={attachments} />
    </aside>
  )
}
