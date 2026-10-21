import type { ImportPreview } from '@/api/crews'
import { Badge, type Tone } from '@/components/ui/Layout'

const tone: Record<string, Tone> = { valid: 'green', invalid: 'red', duplicate: 'yellow' }
const label: Record<string, string> = { valid: 'Valid', invalid: 'Invalid', duplicate: 'Duplicate (skipped)' }

export function ImportPreviewTable({ preview }: { preview: ImportPreview }) {
  return (
    <div className="space-y-3">
      <dl className="flex flex-wrap gap-4 text-sm" aria-label="Validation summary">
        <div>
          <dt className="inline text-slate-500">Rows: </dt>
          <dd className="inline font-semibold">{preview.total}</dd>
        </div>
        <div>
          <dt className="inline text-slate-500">Valid: </dt>
          <dd className="inline font-semibold text-green-700">{preview.valid}</dd>
        </div>
        <div>
          <dt className="inline text-slate-500">Invalid: </dt>
          <dd className="inline font-semibold text-red-700">{preview.invalid}</dd>
        </div>
        <div>
          <dt className="inline text-slate-500">Duplicates: </dt>
          <dd className="inline font-semibold text-yellow-700">{preview.duplicate}</dd>
        </div>
      </dl>
      <div className="max-h-[28rem] overflow-auto rounded-md ring-1 ring-slate-200">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="sticky top-0 bg-slate-50">
            <tr>
              <th scope="col" className="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase">
                Line
              </th>
              <th scope="col" className="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase">
                Status
              </th>
              <th scope="col" className="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase">
                Name
              </th>
              <th scope="col" className="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase">
                Rank
              </th>
              <th scope="col" className="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase">
                Passport
              </th>
              <th scope="col" className="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase">
                Problems
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {preview.rows.map((r) => (
              <tr key={r.line} className={r.status === 'valid' ? undefined : 'bg-red-50/40'}>
                <td className="px-3 py-1.5 font-mono">{r.line}</td>
                <td className="px-3 py-1.5">
                  <Badge tone={tone[r.status] ?? 'neutral'}>{label[r.status] ?? r.status}</Badge>
                </td>
                <td className="px-3 py-1.5">{r.values.full_name ?? ''}</td>
                <td className="px-3 py-1.5">{r.values.applied_rank ?? ''}</td>
                <td className="px-3 py-1.5">{r.values.passport_number ?? ''}</td>
                <td className="px-3 py-1.5 text-xs text-red-700">
                  {r.errors ? (
                    <ul>
                      {Object.entries(r.errors).map(([field, msg]) => (
                        <li key={field}>
                          <span className="font-semibold">{field}</span>: {msg}
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
