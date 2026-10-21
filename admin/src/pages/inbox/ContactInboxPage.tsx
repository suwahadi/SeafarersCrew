import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { useState } from 'react'
import { contactMessagesApi, UNREAD_COUNT_KEY, type ContactMessage, type ContactMessagePatch, type InboxStatus } from '@/api/contactMessages'
import { errorMessage } from '@/api/client'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { Tabs } from '@/components/ui/Tabs'
import { useToast } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/Layout'
import { formatDateTime } from '@/lib/format'
import { openPatch } from '@/lib/inbox'
import { InboxMessagePanel } from './InboxMessagePanel'

const TABS: { id: InboxStatus; label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'unread', label: 'Unread' },
  { id: 'read', label: 'Read' },
  { id: 'archived', label: 'Archived' },
]

function isTab(value: string): value is InboxStatus {
  return TABS.some((t) => t.id === value)
}

function weight(m: ContactMessage): string {
  return m.is_read ? '' : 'font-semibold text-slate-900'
}

const columns: ColumnDef<ContactMessage>[] = [
  {
    id: 'name',
    accessorFn: (r) => r.name,
    header: 'From',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => (
      <span className={weight(row.original)}>
        {row.original.is_read ? null : <span className="sr-only">Unread: </span>}
        {row.original.name}
      </span>
    ),
  },
  { id: 'subject', accessorFn: (r) => r.subject, header: 'Subject', enableSorting: false, cell: ({ row }) => <span className={weight(row.original)}>{row.original.subject}</span> },
  { id: 'email', accessorFn: (r) => r.email, header: 'Email', enableSorting: false, cell: ({ row }) => <span className={weight(row.original)}>{row.original.email}</span> },
  {
    id: 'created_at',
    accessorFn: (r) => r.created_at,
    header: 'Received',
    enableSorting: false,
    cell: ({ row }) => <span className={weight(row.original)}>{formatDateTime(row.original.created_at)}</span>,
  },
]

export default function ContactInboxPage() {
  const qc = useQueryClient()
  const toast = useToast()
  const [status, setStatus] = useState<InboxStatus>('all')
  const [selected, setSelected] = useState<ContactMessage | null>(null)

  const patch = useMutation({
    mutationFn: ({ id, body }: { id: string; body: ContactMessagePatch }) => contactMessagesApi.patch(id, body),
    onSuccess: (saved, { body }) => {
      setSelected((cur) => (cur?.id === saved.id ? saved : cur))
      void qc.invalidateQueries({ queryKey: ['grid', 'contact-messages'] })
      void qc.invalidateQueries({ queryKey: UNREAD_COUNT_KEY })
      if (body.is_archived !== undefined) toast.success(saved.is_archived ? 'Message archived.' : 'Message restored.')
    },
    onError: (err) => toast.error(errorMessage(err)),
  })

  const open = (message: ContactMessage) => {
    setSelected(message)
    const body = openPatch(message)
    if (body) patch.mutate({ id: message.id, body })
  }

  return (
    <>
      <PageHeader title="Contact Inbox" description="Messages sent through the website contact form." />
      <div className="mb-3">
        <Tabs
          tabs={TABS}
          active={status}
          onChange={(id) => {
            if (isTab(id)) {
              setStatus(id)
              setSelected(null)
            }
          }}
        />
      </div>
      <div className="grid gap-4 xl:grid-cols-[1fr_26rem]">
        <DataGrid<ContactMessage>
          id="contact-messages"
          columns={columns}
          filters={{ status }}
          searchPlaceholder="Search name, email or subject…"
          rowKey={(r) => r.id}
          onRowClick={open}
          emptyMessage="No messages."
          fetchPage={async (params, signal) => {
            const res = await contactMessagesApi.list(params, signal)
            return { rows: res.data, total: res.meta.total }
          }}
        />
        {selected ? (
          <InboxMessagePanel
            message={selected}
            busy={patch.isPending}
            onPatch={(body) => patch.mutate({ id: selected.id, body })}
            onClose={() => setSelected(null)}
          />
        ) : null}
      </div>
    </>
  )
}
