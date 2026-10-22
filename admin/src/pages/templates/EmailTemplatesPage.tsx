import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router'
import { errorMessage } from '@/api/client'
import { EMAIL_TEMPLATES_KEY, emailTemplatesApi } from '@/api/templates'
import { Icon } from '@/components/ui/Icon'
import { EmptyState, PageHeader } from '@/components/ui/Layout'
import { Spinner } from '@/components/ui/Spinner'
import { formatDateTime } from '@/lib/format'

export default function EmailTemplatesPage() {
  const templates = useQuery({ queryKey: EMAIL_TEMPLATES_KEY, queryFn: ({ signal }) => emailTemplatesApi.list(signal) })

  return (
    <div>
      <PageHeader title="Email templates" description="Edit the subject and body of every email the system sends." />
      {templates.isPending ? (
        <div className="flex justify-center p-8">
          <Spinner />
        </div>
      ) : templates.isError ? (
        <p role="alert" className="text-sm text-red-700">
          {errorMessage(templates.error)}
        </p>
      ) : templates.data.length === 0 ? (
        <EmptyState title="No email templates" />
      ) : (
        <div className="relative overflow-x-auto rounded-lg bg-white shadow-sm ring-1 ring-slate-200">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-200 bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
              <tr>
                <th scope="col" className="px-4 py-2">Template</th>
                <th scope="col" className="px-4 py-2">Subject</th>
                <th scope="col" className="px-4 py-2">Updated</th>
                <th scope="col" className="px-4 py-2"><span className="sr-only">Actions</span></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {templates.data.map((t) => (
                <tr key={t.code}>
                  <td className="px-4 py-2 font-medium text-slate-900">{t.name}</td>
                  <td className="px-4 py-2 text-slate-600">{t.subject}</td>
                  <td className="px-4 py-2 text-slate-600">{formatDateTime(t.updated_at)}</td>
                  <td className="px-4 py-2 text-right">
                    <Link
                      to={`/email-templates/${t.code}`}
                      aria-label={`Edit ${t.name}`}
                      title={`Edit ${t.name}`}
                      className="inline-flex size-9 items-center justify-center rounded-md text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                    >
                      <Icon name="edit" className="size-[1.125rem]" />
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
