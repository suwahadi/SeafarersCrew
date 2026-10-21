import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useBlocker, useNavigate, useParams, useSearchParams } from 'react-router'
import { ApiError, errorMessage } from '@/api/client'
import { crewKeys, crewsApi, type Crew, type StatusAction } from '@/api/crews'
import { filesApi } from '@/api/files'
import { Button } from '@/components/ui/Button'
import { EmptyState } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { Tabs, type TabItem } from '@/components/ui/Tabs'
import { useToast } from '@/components/ui/Toast'
import { ComplianceBadge, StatusBadge } from './badges'
import { StatusActionDialog, STATUS_ACTIONS } from './StatusActionDialog'
import { AssignmentTab } from './tabs/AssignmentTab'
import { BankTab } from './tabs/BankTab'
import { ComplianceTab } from './tabs/ComplianceTab'
import { DocumentsTab } from './tabs/DocumentsTab'
import { ExperiencesTab } from './tabs/ExperiencesTab'
import { KinTab } from './tabs/KinTab'
import { ContactTab, PersonalTab } from './tabs/ProfileTabs'

const TAB_ITEMS = [
  { id: 'personal', label: 'Personal Data' },
  { id: 'contact', label: 'Contact & Address' },
  { id: 'bank', label: 'Bank Information' },
  { id: 'kin', label: 'Next of Kin' },
  { id: 'documents', label: 'Documents' },
  { id: 'certificates', label: 'Certificates (STCW)' },
  { id: 'experiences', label: 'Experiences' },
  { id: 'assignment', label: 'Current Assignment' },
  { id: 'compliance', label: 'Compliance Status' },
] as const

type TabId = (typeof TAB_ITEMS)[number]['id']

function isTabId(v: string | null): v is TabId {
  return TAB_ITEMS.some((t) => t.id === v)
}

function Photo({ fileId, name }: { fileId: string | null; name: string }) {
  const url = useQuery({
    queryKey: ['crews', 'photo', fileId],
    queryFn: () => filesApi.url(fileId ?? ''),
    enabled: fileId !== null,
    staleTime: 4 * 60_000,
  })
  return (
    <div className="h-20 w-20 shrink-0 overflow-hidden rounded-lg bg-slate-100 ring-1 ring-slate-200">
      {url.data ? (
        <img src={url.data.url} alt={`Photo of ${name}`} width={80} height={80} className="h-20 w-20 object-cover" />
      ) : (
        <div aria-hidden className="flex h-full items-center justify-center text-2xl font-semibold text-slate-400">
          {name.slice(0, 1).toUpperCase()}
        </div>
      )}
    </div>
  )
}

interface HeaderProps {
  crew: Crew
  rankName: string
  compliant: boolean | undefined
  onAction: (a: StatusAction) => void
  onDelete: () => void
}

function CrewHeader({ crew, rankName, compliant, onAction, onDelete }: HeaderProps) {
  const status = crew.status.code
  const blacklisted = status === 'blacklisted'
  return (
    <header className="mb-6 flex flex-wrap items-start justify-between gap-4 rounded-lg bg-white p-4 shadow-sm ring-1 ring-slate-200">
      <div className="flex items-center gap-4">
        <Photo fileId={crew.photo_file_id} name={crew.full_name} />
        <div className="space-y-1">
          <h1 className="text-xl font-semibold text-navy-800">{crew.full_name}</h1>
          <p className="text-sm text-slate-600">
            Crew ID <span className="font-mono">{crew.id}</span>
            {rankName ? <> · {rankName}</> : null}
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge status={crew.status} />
            {compliant !== undefined ? <ComplianceBadge compliant={compliant} /> : null}
            {crew.status_reason ? <span className="text-xs text-slate-500">{crew.status_reason}</span> : null}
          </div>
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Link
          to={`/documents/new?kind=cv&related_id=${crew.id}`}
          className="inline-flex h-9 items-center rounded-md bg-white px-4 text-sm font-medium text-slate-800 ring-1 ring-inset ring-slate-300 hover:bg-slate-50"
        >
          Generate CV
        </Link>
        {status !== 'medical_leave' && status !== 'on_board' && !blacklisted ? (
          <Button variant="secondary" onClick={() => onAction('medical_leave')}>
            {STATUS_ACTIONS.medical_leave.label}
          </Button>
        ) : null}
        {status !== 'available' && status !== 'on_board' ? (
          <Button variant="secondary" onClick={() => onAction('ready')}>
            {STATUS_ACTIONS.ready.label}
          </Button>
        ) : null}
        {!blacklisted && status !== 'on_board' ? (
          <Button variant="secondary" onClick={() => onAction('blacklist')}>
            {STATUS_ACTIONS.blacklist.label}
          </Button>
        ) : null}
        <Button variant="danger" onClick={onDelete}>
          Delete crew
        </Button>
      </div>
    </header>
  )
}

function CrewDetail({ crew }: { crew: Crew }) {
  const qc = useQueryClient()
  const toast = useToast()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const requested = params.get('tab')
  const active: TabId = isTabId(requested) ? requested : 'personal'
  const [dirty, setDirty] = useState<Record<string, boolean>>({})
  const [action, setAction] = useState<StatusAction | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)

  const compliance = useQuery({
    queryKey: [...crewKeys.tab(crew.id, 'compliance'), '', ''],
    queryFn: () => crewsApi.compliance(crew.id, {}),
  })

  const assignment = useQuery({ queryKey: crewKeys.tab(crew.id, 'assignment'), queryFn: () => crewsApi.currentAssignment(crew.id) })

  const setPersonalDirty = useCallback((v: boolean) => setDirty((d) => (d.personal === v ? d : { ...d, personal: v })), [])
  const setContactDirty = useCallback((v: boolean) => setDirty((d) => (d.contact === v ? d : { ...d, contact: v })), [])
  const anyDirty = Object.values(dirty).some(Boolean)

  const blocker = useBlocker(({ currentLocation, nextLocation }) => anyDirty && currentLocation.pathname !== nextLocation.pathname)

  useEffect(() => {
    if (!anyDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [anyDirty])

  const statusMutation = useMutation({
    mutationFn: ({ a, reason }: { a: StatusAction; reason: string }) => crewsApi.setStatus(crew.id, a, reason),
    onSuccess: (updated) => {
      qc.setQueryData(crewKeys.detail(crew.id), updated)
      void qc.invalidateQueries({ queryKey: ['grid', 'crews'] })
      toast.success('Status updated.')
      setAction(null)
    },
    onError: (err) => setActionError(errorMessage(err)),
  })

  const remove = useMutation({
    mutationFn: () => crewsApi.remove(crew.id),
    onSuccess: () => {
      qc.removeQueries({ queryKey: crewKeys.detail(crew.id) })
      void qc.invalidateQueries({ queryKey: ['grid', 'crews'] })
      toast.success('Crew deleted.')
      setDirty({})
      void navigate('/crews', { replace: true })
    },
    onError: (err) => {
      toast.error(errorMessage(err))
      setDeleting(false)
    },
  })

  const tabs: TabItem[] = useMemo(() => TAB_ITEMS.map((t) => ({ id: t.id, label: t.label, dirty: dirty[t.id] === true })), [dirty])

  return (
    <>
      <CrewHeader
        crew={crew}
        rankName={assignment.data?.rank.name ?? crew.applied_rank_name}
        compliant={compliance.data?.compliant}
        onAction={(a) => {
          setActionError(null)
          setAction(a)
        }}
        onDelete={() => setDeleting(true)}
      />
      <Tabs tabs={tabs} active={active} onChange={(id) => setParams({ tab: id }, { replace: true })} />
      <div role="tabpanel" className="mt-4">
        <div hidden={active !== 'personal'}>
          <PersonalTab crew={crew} onDirtyChange={setPersonalDirty} />
        </div>
        <div hidden={active !== 'contact'}>
          <ContactTab crew={crew} onDirtyChange={setContactDirty} />
        </div>
        {active === 'bank' ? <BankTab crewId={crew.id} /> : null}
        {active === 'kin' ? <KinTab crewId={crew.id} /> : null}
        {active === 'documents' ? <DocumentsTab crewId={crew.id} kind="documents" /> : null}
        {active === 'certificates' ? <DocumentsTab crewId={crew.id} kind="certificates" /> : null}
        {active === 'experiences' ? <ExperiencesTab crewId={crew.id} /> : null}
        {active === 'assignment' ? <AssignmentTab crewId={crew.id} /> : null}
        {active === 'compliance' ? <ComplianceTab crewId={crew.id} /> : null}
      </div>

      <StatusActionDialog
        action={action}
        loading={statusMutation.isPending}
        error={actionError}
        onConfirm={(a, reason) => statusMutation.mutate({ a, reason })}
        onClose={() => setAction(null)}
      />
      <ConfirmDialog
        open={deleting}
        title="Delete crew"
        message={`Delete ${crew.full_name} (${crew.id}) and all related records? This cannot be undone.`}
        confirmLabel="Delete crew"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onClose={() => setDeleting(false)}
      />
      <ConfirmDialog
        open={blocker.state === 'blocked'}
        title="Discard unsaved changes?"
        message="You have unsaved changes on this crew record. Leaving now will discard them."
        confirmLabel="Discard and leave"
        danger
        onConfirm={() => blocker.state === 'blocked' && blocker.proceed()}
        onClose={() => blocker.state === 'blocked' && blocker.reset()}
      />
    </>
  )
}

export default function CrewDetailPage() {
  const { id = '' } = useParams()
  const crew = useQuery({ queryKey: crewKeys.detail(id), queryFn: () => crewsApi.get(id) })

  if (crew.isPending) return <p className="text-sm text-slate-500">Loading crew…</p>
  if (!crew.data) {
    const missing = crew.error instanceof ApiError && crew.error.status === 404
    return (
      <EmptyState title={missing ? 'Crew not found' : 'Could not load the crew record'}>
        {missing ? 'No crew member has this ID.' : errorMessage(crew.error)}{' '}
        <Link to="/crews" className="text-navy-700 hover:underline">
          Back to the crew list
        </Link>
      </EmptyState>
    )
  }
  return <CrewDetail key={crew.data.id} crew={crew.data} />
}
