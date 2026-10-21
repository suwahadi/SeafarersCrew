import { useQuery } from '@tanstack/react-query'
import { errorMessage } from '@/api/client'
import { emailApi, type EmailRelatedType } from '@/api/email'
import { formatDateTime } from '@/lib/format'
import { EmailStatusBadge } from './EmailStatusBadge'

export const relatedEmailKey = (relatedType: string, relatedId: string) => ['email-logs', relatedType, relatedId] as const

export function RelatedEmailLog({ relatedType, relatedId }: { relatedType: EmailRelatedType; relatedId: string }) {
  const logs = useQuery({
    queryKey: relatedEmailKey(relatedType, relatedId),
    queryFn: () => emailApi.logs({ related_type: relatedType, related_id: relatedId, per_page: 10 }).then((r) => r.data),
  })
  if (logs.isError) return <p role="alert" className="text-sm text-red-700">{errorMessage(logs.error)}</p>
  const rows = logs.data ?? []
  if (logs.isSuccess && rows.length === 0) return <p className="text-sm text-slate-500">No emails sent yet.</p>
  return (
    <ul className="divide-y divide-slate-100 text-sm">
      {rows.map((l) => (
        <li key={l.id} className="py-2">
          <div className="flex items-center justify-between gap-2">
            <span className="font-medium text-slate-800">{l.subject}</span>
            <EmailStatusBadge status={l.status} />
          </div>
          <p className="text-xs text-slate-500">
            To {l.to.join(', ')} · {formatDateTime(l.sent_at ?? l.created_at)}
          </p>
          {l.error ? <p className="text-xs text-red-700">{l.error}</p> : null}
        </li>
      ))}
    </ul>
  )
}
