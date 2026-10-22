import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { useState } from 'react'
import { errorMessage } from '@/api/client'
import { emailApi, type EmailLog } from '@/api/email'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { EmailStatusBadge } from '@/components/email/EmailStatusBadge'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { filterClass } from '@/components/ui/Field'
import { PageHeader } from '@/components/ui/Layout'
import { useToast } from '@/components/ui/Toast'
import { formatDateTime, titleCase } from '@/lib/format'

export default function EmailLogPage() {
  const qc = useQueryClient()
  const toast = useToast()
  const [status, setStatus] = useState('')
  const [relatedType, setRelatedType] = useState('')
  const [resendingId, setResendingId] = useState<string | null>(null)

  const resend = useMutation({
    mutationFn: (id: string) => emailApi.resend(id),
    onMutate: (id) => setResendingId(id),
    onSuccess: () => {
      toast.success('Email queued for another attempt.')
      void qc.invalidateQueries({ queryKey: ['grid', 'email-logs'] })
    },
    onError: (err) => toast.error(errorMessage(err)),
    onSettled: () => setResendingId(null),
  })

  const columns: ColumnDef<EmailLog>[] = [
    { id: 'created_at', header: 'Queued', accessorFn: (r) => formatDateTime(r.created_at), enableSorting: false },
    { id: 'to', header: 'To', accessorFn: (r) => r.to.join(', '), enableSorting: false },
    { id: 'subject', header: 'Subject', accessorFn: (r) => r.subject, enableSorting: false },
    { id: 'template', header: 'Template', accessorFn: (r) => r.template_code || '—', enableSorting: false },
    { id: 'related', header: 'Related', accessorFn: (r) => (r.related_type ? titleCase(r.related_type) : '—'), enableSorting: false },
    { id: 'status', header: 'Status', cell: ({ row }) => <EmailStatusBadge status={row.original.status} />, enableSorting: false },
    { id: 'attempts', header: 'Attempts', accessorFn: (r) => r.attempts, enableSorting: false },
    {
      id: 'error',
      header: 'Error',
      cell: ({ row }) => (row.original.error ? <span className="block max-w-xs truncate text-red-700" title={row.original.error}>{row.original.error}</span> : '—'),
      enableSorting: false,
    },
    {
      id: 'actions',
      header: 'Actions',
      enableHiding: false,
      cell: ({ row }) =>
        row.original.status === 'failed' ? (
          <Button size="sm" variant="secondary" collapse icon={<Icon name="refresh" />} loading={resendingId === row.original.id} onClick={() => resend.mutate(row.original.id)}>
            Resend
          </Button>
        ) : null,
    },
  ]

  return (
    <>
      <PageHeader title="Email log" description="Every email queued by the system with its delivery status." />
      <div className="filter-bar mb-3" role="group" aria-label="Filters">
        <select aria-label="Status" className={filterClass} value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">All statuses</option>
          <option value="queued">Queued</option>
          <option value="sent">Sent</option>
          <option value="failed">Failed</option>
        </select>
        <select aria-label="Related to" className={filterClass} value={relatedType} onChange={(e) => setRelatedType(e.target.value)}>
          <option value="">All records</option>
          <option value="contract">Contract</option>
          <option value="quotation">Quotation</option>
          <option value="crew">Crew</option>
          <option value="generated_document">Generated document</option>
          <option value="applicant">Applicant</option>
          <option value="contact_message">Contact message</option>
        </select>
      </div>
      <DataGrid<EmailLog>
        id="email-logs"
        columns={columns}
        filters={{ status, related_type: relatedType }}
        searchPlaceholder="Filter this page by recipient or subject…"
        rowKey={(r) => r.id}
        fetchPage={async (params, signal) => {
          const res = await emailApi.logs(params, signal)
          const needle = (params.q ?? '').toLowerCase()
          const rows = needle === '' ? res.data : res.data.filter((l) => l.subject.toLowerCase().includes(needle) || l.to.some((t) => t.includes(needle)))
          return { rows, total: res.meta.total }
        }}
      />
    </>
  )
}
