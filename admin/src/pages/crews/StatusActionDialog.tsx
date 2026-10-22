import { useState } from 'react'
import { type StatusAction } from '@/api/crews'
import { Button } from '@/components/ui/Button'
import { TextAreaField } from '@/components/ui/Field'
import { Modal } from '@/components/ui/Modal'

export const STATUS_ACTIONS: Record<StatusAction, { label: string; title: string; message: string; reasonRequired: boolean; danger: boolean }> = {
  medical_leave: {
    label: 'Medical leave',
    title: 'Set medical leave',
    message: 'The crew member will be marked as on medical leave and cannot be assigned until marked ready.',
    reasonRequired: false,
    danger: false,
  },
  ready: {
    label: 'Mark ready',
    title: 'Mark ready',
    message: 'The crew member will be marked as available for assignment.',
    reasonRequired: false,
    danger: false,
  },
  blacklist: {
    label: 'Blacklist',
    title: 'Blacklist crew member',
    message: 'A blacklisted crew member cannot be assigned to contracts. A reason is required.',
    reasonRequired: true,
    danger: true,
  },
}

interface Props {
  action: StatusAction | null
  loading: boolean
  error?: string | null
  onConfirm: (action: StatusAction, reason: string) => void
  onClose: () => void
}

function DialogBody({ action, loading, error, onConfirm, onClose }: Props & { action: StatusAction }) {
  const cfg = STATUS_ACTIONS[action]
  const [reason, setReason] = useState('')
  const [problem, setProblem] = useState<string | null>(null)

  const submit = () => {
    const trimmed = reason.trim()
    if (cfg.reasonRequired && trimmed === '') {
      setProblem('A reason is required.')
      return
    }
    setProblem(null)
    onConfirm(action, trimmed)
  }

  return (
    <div className="space-y-3">
      <p className="text-sm text-slate-700">{cfg.message}</p>
      {error ? (
        <p role="alert" className="rounded-md bg-red-50 p-2 text-sm text-red-700">
          {error}
        </p>
      ) : null}
      <TextAreaField
        label={cfg.reasonRequired ? 'Reason' : 'Reason (optional)'}
        required={cfg.reasonRequired}
        rows={3}
        maxLength={300}
        value={reason}
        error={problem ?? undefined}
        onChange={(e) => setReason(e.target.value)}
      />
      <div className="flex justify-end gap-2 pt-2 max-sm:*:flex-1">
        <Button variant="secondary" onClick={onClose} disabled={loading}>
          Cancel
        </Button>
        <Button variant={cfg.danger ? 'danger' : 'primary'} loading={loading} onClick={submit}>
          {cfg.label}
        </Button>
      </div>
    </div>
  )
}

export function StatusActionDialog(props: Props) {
  const { action, onClose } = props
  return (
    <Modal open={action !== null} title={action ? STATUS_ACTIONS[action].title : ''} onClose={onClose}>
      {action ? <DialogBody {...props} action={action} /> : null}
    </Modal>
  )
}
