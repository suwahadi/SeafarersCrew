import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage, fieldErrors } from '@/api/client'
import { emailApi, type EmailRelatedType } from '@/api/email'
import { filesApi } from '@/api/files'
import { checkRecipients, recipientError } from '@/lib/email'
import { formatBytes } from '@/lib/format'
import { Button } from '../ui/Button'
import { SelectField, TextAreaField, TextField } from '../ui/Field'
import { Modal } from '../ui/Modal'
import { useToast } from '../ui/Toast'

export interface EmailAttachment {
  fileId: string
  name: string
}

interface EmailSendDialogProps {
  open: boolean
  onClose: () => void
  relatedType?: EmailRelatedType
  relatedId?: string
  defaultTemplate?: string
  defaultTo?: string[]
  attachments?: EmailAttachment[]
}

const DEFAULT_TEMPLATE: Partial<Record<EmailRelatedType, string>> = {
  contract: 'send_contract',
  quotation: 'send_quotation',
  crew: 'send_cv',
}

export function EmailSendDialog(props: EmailSendDialogProps) {
  if (!props.open) return null
  return <EmailSendForm {...props} />
}

function EmailSendForm({ onClose, relatedType, relatedId, defaultTemplate, defaultTo = [], attachments = [] }: EmailSendDialogProps) {
  const toast = useToast()
  const qc = useQueryClient()
  const [templateCode, setTemplateCode] = useState(defaultTemplate ?? (relatedType ? DEFAULT_TEMPLATE[relatedType] : undefined) ?? '')
  const [to, setTo] = useState(defaultTo.join(', '))
  const [cc, setCc] = useState('')
  const [subject, setSubject] = useState<string | null>(null)
  const [body, setBody] = useState<string | null>(null)
  const [extra, setExtra] = useState<EmailAttachment[]>([])
  const [selected, setSelected] = useState<string[]>(attachments.map((a) => a.fileId))
  const [touched, setTouched] = useState(false)
  const [serverFields, setServerFields] = useState<Record<string, string>>({})

  const templates = useQuery({ queryKey: ['email-templates'], queryFn: emailApi.templates, staleTime: 5 * 60_000 })
  const template = useQuery({
    queryKey: ['email-template', templateCode],
    queryFn: () => emailApi.template(templateCode),
    enabled: templateCode !== '',
  })
  const subjectValue = subject ?? template.data?.subject ?? ''
  const bodyValue = body ?? template.data?.body_html ?? ''

  const upload = useMutation({
    mutationFn: (file: File) => filesApi.upload(file, 'crew_document'),
    onSuccess: (f) => {
      setExtra((list) => [...list, { fileId: f.id, name: `${f.original_name} (${formatBytes(f.size_stored)})` }])
      setSelected((list) => [...list, f.id])
    },
    onError: (err) => toast.error(errorMessage(err)),
  })

  const send = useMutation({
    mutationFn: () =>
      emailApi.send({
        template_code: templateCode,
        to: checkRecipients(to).valid,
        cc: checkRecipients(cc).valid,
        subject: subjectValue,
        body_html: bodyValue,
        related_type: relatedType,
        related_id: relatedId,
        attachment_file_ids: selected,
      }),
    onSuccess: () => {
      toast.success('Email queued. Its delivery status appears in the email log.')
      void qc.invalidateQueries({ queryKey: ['grid', 'email-logs'] })
      void qc.invalidateQueries({ queryKey: ['email-logs'] })
      onClose()
    },
    onError: (err) => {
      setServerFields(fieldErrors(err))
      toast.error(errorMessage(err))
    },
  })

  const errors = recipientError(to, cc)
  const allAttachments = [...attachments, ...extra]
  const canSend = !errors.to && !errors.cc && subjectValue.trim() !== '' && bodyValue.trim() !== ''

  return (
    <Modal
      open
      size="lg"
      title="Send by email"
      onClose={onClose}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={send.isPending}>
            Cancel
          </Button>
          <Button
            onClick={() => {
              setTouched(true)
              if (canSend) send.mutate()
            }}
            loading={send.isPending}
          >
            Send email
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <SelectField
          label="Template"
          value={templateCode}
          onChange={(e) => {
            setTemplateCode(e.target.value)
            setSubject(null)
            setBody(null)
          }}
          placeholder="No template"
          options={(templates.data ?? []).filter((t) => t.code.startsWith('send_')).map((t) => ({ value: t.code, label: t.name }))}
        />
        <TextField
          label="To"
          required
          value={to}
          onChange={(e) => setTo(e.target.value)}
          onBlur={() => setTouched(true)}
          hint="Separate addresses with commas."
          error={touched ? (errors.to ?? serverFields.to) : serverFields.to}
        />
        <TextField label="CC" value={cc} onChange={(e) => setCc(e.target.value)} error={errors.cc ?? serverFields.cc} />
        <TextField label="Subject" required value={subjectValue} onChange={(e) => setSubject(e.target.value)} error={touched && subjectValue.trim() === '' ? 'The subject is required.' : serverFields.subject} />
        <TextAreaField
          label="Message (HTML)"
          rows={10}
          value={bodyValue}
          onChange={(e) => setBody(e.target.value)}
          hint="Tags such as {crew_name} or {contract_id} are filled from the record when the email is sent."
          error={touched && bodyValue.trim() === '' ? 'The message is required.' : serverFields.body_html}
        />
        <fieldset className="space-y-2">
          <legend className="text-sm font-medium text-slate-700">Attachments</legend>
          {allAttachments.length === 0 ? <p className="text-sm text-slate-500">No attachments.</p> : null}
          {allAttachments.map((a) => (
            <label key={a.fileId} className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={selected.includes(a.fileId)}
                onChange={(e) => setSelected((list) => (e.target.checked ? [...list, a.fileId] : list.filter((x) => x !== a.fileId)))}
              />
              {a.name}
            </label>
          ))}
          <label className="inline-flex cursor-pointer items-center gap-2 text-sm text-navy-700">
            <input
              type="file"
              className="sr-only"
              accept=".pdf,.jpg,.jpeg,.png,.webp"
              onChange={(e) => {
                const file = e.target.files?.[0]
                if (file) upload.mutate(file)
                e.target.value = ''
              }}
            />
            <span className="underline">{upload.isPending ? 'Uploading…' : 'Add a file'}</span>
          </label>
          {serverFields.attachment_file_ids ? <p className="text-xs text-red-600">{serverFields.attachment_file_ids}</p> : null}
        </fieldset>
      </div>
    </Modal>
  )
}
