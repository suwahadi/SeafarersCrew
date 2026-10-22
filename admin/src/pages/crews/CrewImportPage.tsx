import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { Link } from 'react-router'
import { errorMessage } from '@/api/client'
import { crewKeys, crewsApi, type ImportPreview, type ImportReport } from '@/api/crews'
import { waitForJob } from '@/api/jobs'
import { Button } from '@/components/ui/Button'
import { Card, PageHeader } from '@/components/ui/Layout'
import { ImportPreviewTable } from './ImportPreviewTable'
import { downloadBlob } from './useJobDownload'

export default function CrewImportPage() {
  const qc = useQueryClient()
  const input = useRef<HTMLInputElement>(null)
  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [report, setReport] = useState<ImportReport | null>(null)

  const template = useMutation({
    mutationFn: crewsApi.importTemplate,
    onSuccess: (blob) => downloadBlob(blob, 'crew-import-template.csv'),
  })
  const validate = useMutation({
    mutationFn: crewsApi.importValidate,
    onSuccess: (p) => {
      setPreview(p)
      setReport(null)
    },
  })
  const commit = useMutation({
    mutationFn: async (token: string) => {
      const queued = await crewsApi.importCommit(token)
      const job = await waitForJob(queued.id)
      if (job.status === 'failed') throw new Error(job.error ?? 'The import failed.')
      return job.result as ImportReport
    },
    onSuccess: (r) => {
      setReport(r)
      setPreview(null)
      void qc.invalidateQueries({ queryKey: ['grid', 'crews'] })
      void qc.invalidateQueries({ queryKey: crewKeys.all })
    },
  })

  const error = template.error ?? validate.error ?? commit.error

  return (
    <>
      <PageHeader title="Import crew" description="Upload a CSV based on the template. Every row is validated before anything is saved." />
      <div className="space-y-4">
        <Card title="1. Prepare the file">
          <p className="mb-3 text-sm text-slate-600">Dates use YYYY-MM-DD, phones use E.164 (+62…), and ranks use the short code or the name.</p>
          <Button variant="secondary" loading={template.isPending} onClick={() => template.mutate()}>
            Download template
          </Button>
        </Card>
        <Card title="2. Upload and validate">
          <input
            ref={input}
            type="file"
            accept=".csv,text/csv"
            aria-label="CSV file"
            className="sr-only"
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) validate.mutate(file)
              e.target.value = ''
            }}
          />
          <Button loading={validate.isPending} onClick={() => input.current?.click()}>
            Choose CSV file
          </Button>
          {preview ? (
            <div className="mt-4 space-y-4">
              <ImportPreviewTable preview={preview} />
              <div className="flex justify-end gap-2 max-sm:*:flex-1">
                <Button variant="secondary" onClick={() => setPreview(null)} disabled={commit.isPending}>
                  Discard
                </Button>
                <Button disabled={preview.valid === 0} loading={commit.isPending} onClick={() => commit.mutate(preview.token)}>
                  Import {preview.valid} valid row{preview.valid === 1 ? '' : 's'}
                </Button>
              </div>
            </div>
          ) : null}
        </Card>
        {error ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {errorMessage(error)}
          </p>
        ) : null}
        {report ? (
          <Card title="3. Result">
            <p className="text-sm">
              Created <strong>{report.created}</strong>, skipped <strong>{report.skipped}</strong>, failed <strong>{report.failed}</strong>.
            </p>
            {report.outcomes.some((o) => o.error) ? (
              <ul className="mt-3 space-y-1 text-sm text-red-700">
                {report.outcomes
                  .filter((o) => o.error)
                  .map((o) => (
                    <li key={o.line}>
                      Line {o.line}: {o.error}
                    </li>
                  ))}
              </ul>
            ) : null}
            <Link to="/crews" className="mt-4 inline-block text-sm text-navy-700 hover:underline">
              Back to the crew list
            </Link>
          </Card>
        ) : null}
      </div>
    </>
  )
}
