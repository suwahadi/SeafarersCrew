import { useQuery } from '@tanstack/react-query'
import { errorMessage } from '@/api/client'
import { documentsApi, KIND_LABELS, type GeneratedDocument, type RelatedType } from '@/api/documents'
import { openFile } from '@/api/files'
import { formatBytes, formatDateTime } from '@/lib/format'
import { Button } from '../ui/Button'
import { useToast } from '../ui/Toast'

interface DocumentHistoryProps {
  relatedType: RelatedType
  relatedId: string
  onSend?: (doc: GeneratedDocument) => void
}

export const historyKey = (relatedType: string, relatedId: string) => ['generated-documents', relatedType, relatedId] as const

export function DocumentHistory({ relatedType, relatedId, onSend }: DocumentHistoryProps) {
  const toast = useToast()
  const history = useQuery({ queryKey: historyKey(relatedType, relatedId), queryFn: () => documentsApi.history(relatedType, relatedId) })

  if (history.isError) return <p role="alert" className="text-sm text-red-700">{errorMessage(history.error)}</p>
  const docs = history.data ?? []
  if (history.isSuccess && docs.length === 0) return <p className="text-sm text-slate-500">No documents generated yet.</p>

  return (
    <ul className="divide-y divide-slate-100 text-sm">
      {docs.map((d) => (
        <li key={d.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
          <div>
            <p className="font-medium text-slate-800">{KIND_LABELS[d.kind]}</p>
            <p className="text-xs text-slate-500">
              {d.template_name} · {formatDateTime(d.created_at)} · {formatBytes(d.file_size)}
            </p>
          </div>
          <div className="flex gap-1">
            <Button variant="secondary" size="sm" onClick={() => void openFile(d.file_id).catch((e: unknown) => toast.error(errorMessage(e)))}>
              Open
            </Button>
            {onSend ? (
              <Button variant="secondary" size="sm" onClick={() => onSend(d)}>
                Email
              </Button>
            ) : null}
          </div>
        </li>
      ))}
    </ul>
  )
}
