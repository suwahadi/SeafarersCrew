import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { applicantsApi, type ApplicantDetail, type Stage, type StageLog } from '@/api/applicants'
import { errorMessage } from '@/api/client'
import { openFile } from '@/api/files'
import { Button } from '@/components/ui/Button'
import { inputClass } from '@/components/ui/Field'
import { Card, DescriptionList, EmptyState, PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { formatBytes, formatDate, formatDateTime } from '@/lib/format'
import { STAGE_LABELS, moveTargets } from '@/lib/pipeline'
import { StageBadge } from './StageBadge'
import { StageMoveDialog } from './StageMoveDialog'
import { useStageMove } from './useStageMove'

export default function ApplicantDetailPage() {
  const { id = '' } = useParams()
  const applicant = useQuery({ queryKey: ['applicant', id], queryFn: () => applicantsApi.get(id) })
  if (applicant.isPending) {
    return (
      <div className="flex justify-center p-12 text-navy-800">
        <Spinner size="lg" label="Loading applicant" />
      </div>
    )
  }
  if (applicant.isError) return <p role="alert" className="text-sm text-red-700">{errorMessage(applicant.error)}</p>
  return <ApplicantDetailView applicant={applicant.data} />
}

export function ApplicantDetailView({ applicant }: { applicant: ApplicantDetail }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const [deleting, setDeleting] = useState(false)
  const [target, setTarget] = useState<Stage | ''>('')
  const move = useStageMove(() => setTarget(''))

  const download = useMutation({
    mutationFn: (fileId: string) => openFile(fileId),
    onError: (err) => toast.error(errorMessage(err)),
  })

  const remove = useMutation({
    mutationFn: () => applicantsApi.remove(applicant.id),
    onSuccess: () => {
      qc.removeQueries({ queryKey: ['applicant', applicant.id] })
      void qc.invalidateQueries({ queryKey: ['applicants'] })
      void qc.invalidateQueries({ queryKey: ['grid', 'applicants'] })
      void qc.invalidateQueries({ queryKey: ['grid', 'vacancies'] })
      toast.success('Applicant deleted.')
      void navigate('/applicants', { replace: true })
    },
    onError: (err) => {
      toast.error(errorMessage(err))
      setDeleting(false)
    },
  })

  const targets = moveTargets(applicant.stage)

  return (
    <>
      <PageHeader
        title={applicant.full_name}
        description={`Application ${applicant.id} · applied ${formatDateTime(applicant.created_at)}`}
        actions={
          <>
            <StageBadge stage={applicant.stage} />
            <Link to="/applicants" className="text-sm text-navy-700 hover:underline">
              Back to applicants
            </Link>
          </>
        }
      />
      <div className="grid gap-6 xl:grid-cols-[1fr_24rem]">
        <div className="space-y-6">
          <Card title="Applicant">
            <DescriptionList
              items={[
                { label: 'Full name', value: applicant.full_name },
                { label: 'Email', value: <a href={`mailto:${applicant.email}`} className="text-navy-700 hover:underline">{applicant.email}</a> },
                { label: 'Phone', value: applicant.phone },
                { label: 'Date of birth', value: formatDate(applicant.date_of_birth) },
                { label: 'Nationality', value: applicant.nationality },
                { label: 'Rank applied', value: applicant.rank_applied?.name ?? '' },
                { label: 'Years of experience', value: applicant.years_experience === null ? '' : String(applicant.years_experience) },
                {
                  label: 'Vacancy',
                  value: (
                    <Link to={`/vacancies/${applicant.vacancy.id}`} className="text-navy-700 hover:underline">
                      {applicant.vacancy.title} ({applicant.vacancy.id})
                    </Link>
                  ),
                },
                { label: 'Consent given', value: formatDateTime(applicant.consent_at) },
                { label: 'IP address', value: applicant.ip_address },
              ]}
            />
          </Card>
          <Card title="Cover message">
            {applicant.message.trim() === '' ? (
              <p className="text-sm text-slate-500">The applicant did not leave a message.</p>
            ) : (
              <p className="text-sm break-words whitespace-pre-wrap text-slate-900">{applicant.message}</p>
            )}
          </Card>
          <Card title="Files">
            {applicant.files.length === 0 ? (
              <EmptyState title="No files were uploaded." />
            ) : (
              <ul className="divide-y divide-slate-100">
                {applicant.files.map((f) => (
                  <li key={f.file_id} className="flex flex-wrap items-center justify-between gap-2 py-2">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-slate-800">
                        {f.name} {f.kind === 'cv' ? <span className="ml-1 rounded bg-navy-50 px-1.5 py-0.5 text-xs font-medium text-navy-800">CV</span> : null}
                      </p>
                      <p className="text-xs text-slate-500">
                        {formatBytes(f.size)} · {f.mime_type} · {formatDateTime(f.created_at)}
                      </p>
                    </div>
                    <Button
                      variant="secondary"
                      size="sm"
                      aria-label={`Download ${f.name}`}
                      loading={download.isPending && download.variables === f.file_id}
                      onClick={() => download.mutate(f.file_id)}
                    >
                      Download
                    </Button>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
        <div className="space-y-6">
          <Card title="Pipeline stage">
            <div className="space-y-3">
              <div className="space-y-1">
                <label htmlFor="move-stage" className="block text-sm font-medium text-slate-700">
                  Move to
                </label>
                <select id="move-stage" className={inputClass} value={target} onChange={(e) => setTarget(targets.find((s) => s === e.target.value) ?? '')}>
                  <option value="">Select a stage</option>
                  {targets.map((s) => (
                    <option key={s} value={s}>
                      {STAGE_LABELS[s]}
                    </option>
                  ))}
                </select>
              </div>
              <Button
                disabled={target === ''}
                onClick={() => {
                  if (target !== '') move.start({ id: applicant.id, name: applicant.full_name, from: applicant.stage, to: target })
                }}
              >
                Change stage
              </Button>
            </div>
          </Card>
          <Card title="Stage history">
            <StageTimeline logs={applicant.stage_logs} />
          </Card>
          <Button variant="danger" onClick={() => setDeleting(true)}>
            Delete applicant
          </Button>
        </div>
      </div>
      {move.request ? (
        <StageMoveDialog
          open
          applicantName={move.request.name}
          from={move.request.from}
          to={move.request.to}
          loading={move.loading}
          error={move.error}
          onConfirm={move.confirm}
          onClose={move.cancel}
        />
      ) : null}
      <ConfirmDialog
        open={deleting}
        title="Delete applicant?"
        message="The application is removed from the pipeline. This cannot be undone from the admin."
        confirmLabel="Delete"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onClose={() => setDeleting(false)}
      />
    </>
  )
}

export function StageTimeline({ logs }: { logs: StageLog[] }) {
  if (logs.length === 0) return <p className="text-sm text-slate-500">No stage changes yet.</p>
  return (
    <ol className="space-y-4 border-l border-slate-200 pl-4">
      {[...logs]
        .sort((a, b) => b.created_at.localeCompare(a.created_at))
        .map((log) => (
          <li key={log.id} className="relative">
            <span aria-hidden className="absolute top-1.5 -left-[1.3rem] h-2 w-2 rounded-full bg-navy-600" />
            <p className="text-sm font-medium text-slate-800">
              {log.from_stage ? `${STAGE_LABELS[log.from_stage]} → ` : 'Applied · '}
              {STAGE_LABELS[log.to_stage]}
            </p>
            <p className="text-xs text-slate-500">
              {formatDateTime(log.created_at)} · {log.admin?.name ?? 'System'}
            </p>
            {log.note ? <p className="mt-1 text-sm break-words whitespace-pre-wrap text-slate-700">{log.note}</p> : null}
          </li>
        ))}
    </ol>
  )
}
