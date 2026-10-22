import type { ColumnDef } from '@tanstack/react-table'
import { useState } from 'react'
import { useNavigate } from 'react-router'
import { errorMessage } from '@/api/client'
import { documentsApi, KIND_LABELS, type DocumentKind, type GeneratedDocument } from '@/api/documents'
import { openFile } from '@/api/files'
import { DataGrid } from '@/components/datagrid/DataGrid'
import { EmailSendDialog } from '@/components/email/EmailSendDialog'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { filterClass } from '@/components/ui/Field'
import { PageHeader } from '@/components/ui/Layout'
import { useToast } from '@/components/ui/Toast'
import { formatBytes, formatDateTime } from '@/lib/format'

const EMAIL_TEMPLATE: Record<DocumentKind, string> = { sea: 'send_contract', quotation: 'send_quotation', cv: 'send_cv' }

export default function DocumentListPage() {
  const navigate = useNavigate()
  const toast = useToast()
  const [kind, setKind] = useState('')
  const [relatedType, setRelatedType] = useState('')
  const [sending, setSending] = useState<GeneratedDocument | null>(null)

  const columns: ColumnDef<GeneratedDocument>[] = [
    { id: 'id', header: 'Document ID', accessorFn: (r) => r.id, enableSorting: false },
    { id: 'kind', header: 'Type', accessorFn: (r) => KIND_LABELS[r.kind], enableSorting: false },
    { id: 'template', header: 'Template', accessorFn: (r) => r.template_name, enableSorting: false },
    { id: 'size', header: 'Size', accessorFn: (r) => formatBytes(r.file_size), enableSorting: false },
    { id: 'created_at', header: 'Generated', accessorFn: (r) => formatDateTime(r.created_at), enableSorting: false },
    {
      id: 'actions',
      header: 'Actions',
      enableHiding: false,
      cell: ({ row }) => (
        <div className="flex gap-1">
          <Button size="sm" variant="secondary" collapse icon={<Icon name="external" />} onClick={() => void openFile(row.original.file_id).catch((e: unknown) => toast.error(errorMessage(e)))}>
            Open
          </Button>
          <Button size="sm" variant="secondary" collapse icon={<Icon name="mail" />} onClick={() => setSending(row.original)}>
            Send by email
          </Button>
        </div>
      ),
    },
  ]

  return (
    <>
      <PageHeader
        title="Generated documents"
        description="PDF contracts, quotations and CVs generated from templates."
        actions={
          <Button icon={<Icon name="file" />} onClick={() => void navigate('/documents/new')}>
            Generate document
          </Button>
        }
      />
      <div className="filter-bar mb-3" role="group" aria-label="Filters">
        <select aria-label="Document type" className={filterClass} value={kind} onChange={(e) => setKind(e.target.value)}>
          <option value="">All types</option>
          {Object.entries(KIND_LABELS).map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
        <select aria-label="Related to" className={filterClass} value={relatedType} onChange={(e) => setRelatedType(e.target.value)}>
          <option value="">All records</option>
          <option value="contract">Contracts</option>
          <option value="quotation">Quotations</option>
          <option value="crew">Crew</option>
        </select>
      </div>
      <DataGrid<GeneratedDocument>
        id="documents"
        columns={columns}
        filters={{ kind, related_type: relatedType }}
        searchPlaceholder="Search by ID or template…"
        rowKey={(r) => r.id}
        fetchPage={async (params) => {
          const all = await documentsApi.history(relatedType === '' ? undefined : (relatedType as 'contract' | 'quotation' | 'crew'))
          const needle = (params.q ?? '').toLowerCase()
          const rows = all.filter(
            (d) => (kind === '' || d.kind === kind) && (needle === '' || d.id.toLowerCase().includes(needle) || d.template_name.toLowerCase().includes(needle)),
          )
          const start = (params.page - 1) * params.per_page
          return { rows: rows.slice(start, start + params.per_page), total: rows.length }
        }}
      />
      {sending ? (
        <EmailSendDialog
          open
          onClose={() => setSending(null)}
          defaultTemplate={EMAIL_TEMPLATE[sending.kind]}
          attachments={[{ fileId: sending.file_id, name: `${KIND_LABELS[sending.kind]} (${sending.id})` }]}
        />
      ) : null}
    </>
  )
}
