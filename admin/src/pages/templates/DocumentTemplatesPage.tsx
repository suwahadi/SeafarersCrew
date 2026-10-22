import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router'
import { ApiError, errorMessage } from '@/api/client'
import { DOCUMENT_TEMPLATES_KEY, documentTemplatesApi, type DocumentKind, type DocumentTemplate } from '@/api/templates'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { Badge, Card, EmptyState, PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { formatDateTime } from '@/lib/format'

export const kindLabels: Record<DocumentKind, string> = { sea: 'Seafarer Employment Agreement', quotation: 'Quotation', cv: 'Crew CV' }
const kinds: DocumentKind[] = ['sea', 'quotation', 'cv']

export default function DocumentTemplatesPage() {
  const qc = useQueryClient()
  const toast = useToast()
  const [deleting, setDeleting] = useState<DocumentTemplate | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const templates = useQuery({ queryKey: DOCUMENT_TEMPLATES_KEY, queryFn: ({ signal }) => documentTemplatesApi.list(undefined, signal) })

  const remove = useMutation({
    mutationFn: (t: DocumentTemplate) => documentTemplatesApi.remove(t.id),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: DOCUMENT_TEMPLATES_KEY })
      toast.success('Template deleted.')
      setDeleting(null)
    },
    onError: (err) => setDeleteError(err instanceof ApiError ? err.message : errorMessage(err)),
  })

  return (
    <div>
      <PageHeader
        title="Document templates"
        description="Layouts used to generate contracts, quotations and CVs as PDF."
        actions={
          <Link
            to="/document-templates/new"
            aria-label="New template"
            title="New template"
            className="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-navy-800 px-4 text-sm font-medium text-white hover:bg-navy-700 max-sm:size-9 max-sm:px-0"
          >
            <Icon name="plus" />
            <span className="max-sm:sr-only">New template</span>
          </Link>
        }
      />
      {templates.isPending ? (
        <div className="flex justify-center p-8">
          <Spinner />
        </div>
      ) : templates.isError ? (
        <p role="alert" className="text-sm text-red-700">
          {errorMessage(templates.error)}
        </p>
      ) : (
        <div className="space-y-6">
          {kinds.map((kind) => {
            const rows = templates.data.filter((t) => t.kind === kind)
            return (
              <Card key={kind} title={kindLabels[kind]}>
                {rows.length === 0 ? (
                  <EmptyState title="No templates of this kind yet" />
                ) : (
                  <ul className="divide-y divide-slate-100">
                    {rows.map((t) => (
                      <li key={t.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 py-2 text-sm">
                        <span className="min-w-0 break-words font-medium text-slate-900">{t.name}</span>
                        {t.is_default ? <Badge tone="green">Default</Badge> : null}
                        <span className="text-slate-500">
                          {t.paper_size.toUpperCase()} · {t.orientation}
                        </span>
                        <span className="text-xs text-slate-500">Updated {formatDateTime(t.updated_at)}</span>
                        <span className="ml-auto flex gap-1">
                          <Link
                            to={`/document-templates/${t.id}`}
                            aria-label={`Edit ${t.name}`}
                            title={`Edit ${t.name}`}
                            className="inline-flex size-9 items-center justify-center rounded-md text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                          >
                            <Icon name="edit" className="size-[1.125rem]" />
                          </Link>
                          <Button
                            size="sm"
                            variant="ghost"
                            icon={<Icon name="trash" />}
                            collapse
                            aria-label={`Delete ${t.name}`}
                            onClick={() => {
                              setDeleteError(null)
                              setDeleting(t)
                            }}
                          >
                            Delete
                          </Button>
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </Card>
            )
          })}
        </div>
      )}
      <ConfirmDialog
        open={deleting !== null}
        title="Delete template"
        danger
        confirmLabel="Delete"
        loading={remove.isPending}
        onClose={() => setDeleting(null)}
        onConfirm={() => {
          if (deleting) remove.mutate(deleting)
        }}
        message={
          deleteError ? (
            <p role="alert" className="text-red-700">
              {deleteError}
            </p>
          ) : (
            <p>Delete {deleting?.name}? Documents already generated are not affected.</p>
          )
        }
      />
    </div>
  )
}
