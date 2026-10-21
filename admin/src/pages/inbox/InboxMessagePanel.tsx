import type { ContactMessage, ContactMessagePatch } from '@/api/contactMessages'
import { Button } from '@/components/ui/Button'
import { Badge, Card } from '@/components/ui/Layout'
import { formatDateTime } from '@/lib/format'
import { archiveTogglePatch, readTogglePatch, replyHref } from '@/lib/inbox'

interface InboxMessagePanelProps {
  message: ContactMessage
  busy: boolean
  onPatch: (patch: ContactMessagePatch) => void
  onClose: () => void
}

export function InboxMessagePanel({ message, busy, onPatch, onClose }: InboxMessagePanelProps) {
  return (
    <Card
      title={message.subject}
      actions={
        <button type="button" onClick={onClose} aria-label="Close message" className="rounded p-1 text-slate-500 hover:bg-slate-100">
          ✕
        </button>
      }
    >
      <article aria-label="Message detail" className="space-y-4">
        <div className="flex flex-wrap items-center gap-2">
          {message.is_read ? <Badge>Read</Badge> : <Badge tone="blue">Unread</Badge>}
          {message.is_archived ? <Badge tone="orange">Archived</Badge> : null}
        </div>
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
          <dt className="text-slate-500">From</dt>
          <dd className="text-slate-900">{message.name}</dd>
          <dt className="text-slate-500">Email</dt>
          <dd className="break-all text-slate-900">{message.email}</dd>
          {message.phone ? (
            <>
              <dt className="text-slate-500">Phone</dt>
              <dd className="text-slate-900">{message.phone}</dd>
            </>
          ) : null}
          <dt className="text-slate-500">Received</dt>
          <dd className="text-slate-900">{formatDateTime(message.created_at)}</dd>
          <dt className="text-slate-500">IP address</dt>
          <dd className="text-slate-900">{message.ip_address || '—'}</dd>
        </dl>
        <p className="text-sm break-words whitespace-pre-wrap text-slate-900">{message.message}</p>
        <div className="flex flex-wrap gap-2">
          <a
            href={replyHref(message)}
            className="inline-flex h-9 items-center justify-center rounded-md bg-navy-800 px-4 text-sm font-medium text-white transition-colors duration-150 ease-out hover:bg-navy-700"
          >
            Reply by email
          </a>
          <Button variant="secondary" disabled={busy} onClick={() => onPatch(readTogglePatch(message))}>
            {message.is_read ? 'Mark as unread' : 'Mark as read'}
          </Button>
          <Button variant="secondary" disabled={busy} onClick={() => onPatch(archiveTogglePatch(message))}>
            {message.is_archived ? 'Unarchive' : 'Archive'}
          </Button>
        </div>
      </article>
    </Card>
  )
}
