import { useId, useState } from 'react'
import type { Stage } from '@/api/applicants'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { NOTE_MAX, STAGE_LABELS, noteRequired, validateStageNote } from '@/lib/pipeline'

interface StageMoveDialogProps {
  open: boolean
  applicantName: string
  from: Stage
  to: Stage
  loading?: boolean
  error?: string | null
  onConfirm: (note: string) => void
  onClose: () => void
}

export function StageMoveDialog({ open, applicantName, from, to, loading = false, error = null, onConfirm, onClose }: StageMoveDialogProps) {
  return (
    <Modal open={open} title={`Move to ${STAGE_LABELS[to]}`} onClose={onClose}>
      {open ? <StageMoveForm key={`${from}-${to}`} applicantName={applicantName} from={from} to={to} loading={loading} error={error} onConfirm={onConfirm} onClose={onClose} /> : null}
    </Modal>
  )
}

function StageMoveForm({ applicantName, from, to, loading, error, onConfirm, onClose }: Omit<StageMoveDialogProps, 'open'>) {
  const [note, setNote] = useState('')
  const [problem, setProblem] = useState<string | null>(null)
  const id = useId()
  const required = noteRequired(to)

  const submit = () => {
    const invalid = validateStageNote(to, note)
    setProblem(invalid)
    if (invalid === null) onConfirm(note.trim())
  }

  const shown = problem ?? error

  return (
    <form
      noValidate
      onSubmit={(e) => {
        e.preventDefault()
        submit()
      }}
      className="space-y-3"
    >
      <p className="text-sm text-slate-700">
        Move <strong>{applicantName}</strong> from {STAGE_LABELS[from]} to {STAGE_LABELS[to]}.
      </p>
      <div className="space-y-1">
        <label htmlFor={id} className="block text-sm font-medium text-slate-700">
          Note
          {required ? <span className="text-red-600"> *</span> : <span className="font-normal text-slate-500"> (optional)</span>}
        </label>
        <textarea
          id={id}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          rows={4}
          maxLength={NOTE_MAX}
          autoFocus
          aria-required={required}
          aria-invalid={shown ? true : undefined}
          aria-describedby={shown ? `${id}-error` : undefined}
          className="block w-full rounded-md border-0 px-3 py-1.5 text-sm ring-1 ring-inset ring-slate-300 focus:ring-2 focus:ring-inset focus:ring-navy-500 aria-[invalid=true]:ring-red-500"
        />
        {shown ? (
          <p id={`${id}-error`} role="alert" className="text-xs text-red-600">
            {shown}
          </p>
        ) : null}
      </div>
      <div className="flex justify-end gap-2">
        <Button variant="secondary" onClick={onClose} disabled={loading}>
          Cancel
        </Button>
        <Button type="submit" loading={loading}>
          Move applicant
        </Button>
      </div>
    </form>
  )
}
