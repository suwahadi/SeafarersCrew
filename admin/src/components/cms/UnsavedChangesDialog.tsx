import type { Blocker } from 'react-router'
import { ConfirmDialog } from '@/components/ui/Modal'

export function UnsavedChangesDialog({ blocker, subject }: { blocker: Blocker; subject: string }) {
  return (
    <ConfirmDialog
      open={blocker.state === 'blocked'}
      title="Discard unsaved changes?"
      message={`You have unsaved changes on this ${subject}. Leaving now will discard them.`}
      confirmLabel="Discard and leave"
      danger
      onConfirm={() => blocker.state === 'blocked' && blocker.proceed()}
      onClose={() => blocker.state === 'blocked' && blocker.reset()}
    />
  )
}
