import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useId, useRef, useState, type ReactNode } from 'react'
import { Link, useNavigate } from 'react-router'
import { errorMessage, fieldErrors } from '@/api/client'
import { crewCvApi, type CvChoice, type CvChoiceKind, type CvDecision, type CvPreview, type CvRow, type CvSection } from '@/api/crewCv'
import { crewKeys } from '@/api/crews'
import { OPTIONS_KEY } from '@/api/options'
import { Button } from '@/components/ui/Button'
import { inputClass } from '@/components/ui/Field'
import { Icon } from '@/components/ui/Icon'
import { Badge, Card } from '@/components/ui/Layout'
import { useToast } from '@/components/ui/Toast'
import { useOptions } from '@/hooks/useOptions'

const kindLabel: Record<CvChoiceKind, string> = { certificate: 'Certificate', document: 'Document', rank: 'Rank', vessel_type: 'Vessel type' }
const sectionLabel: Record<CvSection, string> = { personal: 'Personal data', documents: 'Documents', certificates: 'Certificates', sea_service: 'Sea service' }
const reasonText = { passport: 'same passport number', seaman_book: 'same seaman book number', name_and_birth_date: 'same name and date of birth' } as const

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`
const waiting = (rows: CvRow[]) => rows.filter((r) => (r.choice_keys ?? []).length > 0).length
const needs = (k: number) => (k > 0 ? ` (${k} need${k === 1 ? 's' : ''} a choice)` : '')

function summaryText(p: CvPreview): string {
  return [
    '1 crew',
    plural(p.documents.length, 'document') + needs(waiting(p.documents)),
    plural(p.certificates.length, 'certificate') + needs(waiting(p.certificates)),
    `${p.sea_service.length} sea service${needs(waiting(p.sea_service))}`,
    `${p.next_of_kin.length} next of kin`,
  ].join(' · ')
}

function initialDecisions(choices: CvChoice[]): Record<string, CvDecision> {
  const out: Record<string, CvDecision> = {}
  for (const c of choices) {
    if (c.suggested_id) out[c.key] = { action: 'match', target_id: c.suggested_id }
  }
  return out
}

function encode(d: CvDecision | undefined): string {
  if (!d) return ''
  return d.action === 'match' ? `match:${d.target_id ?? ''}` : d.action
}

function decode(value: string): CvDecision | undefined {
  if (value === 'skip' || value === 'create') return { action: value }
  if (value.startsWith('match:')) return { action: 'match', target_id: value.slice('match:'.length) }
  return undefined
}

interface Group {
  label: string
  items: { id: string; name: string }[]
}

function ChoiceField({ choice, value, groups, onChange }: { choice: CvChoice; value: string; groups: Group[]; onChange: (value: string) => void }) {
  const id = useId()
  return (
    <div className="grid gap-1 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)] sm:items-center sm:gap-4">
      <label htmlFor={id} className="text-sm text-slate-700">
        <span className="font-medium">{kindLabel[choice.kind]} in CV:</span> {choice.text}
      </label>
      <select id={id} value={value} onChange={(e) => onChange(e.target.value)} className={inputClass}>
        <option value="">Choose…</option>
        {groups.map((g) => (
          <optgroup key={g.label} label={g.label}>
            {g.items.map((it) => (
              <option key={it.id} value={`match:${it.id}`}>
                {it.name}
              </option>
            ))}
          </optgroup>
        ))}
        {choice.allow_create ? <option value="create">Create as a new type</option> : null}
        <option value="skip">Skip</option>
      </select>
    </div>
  )
}

interface Column {
  label: string
  render: (row: CvRow) => ReactNode
  wrap?: boolean
}

function CvTable({ title, rows, columns }: { title: string; rows: CvRow[]; columns: Column[] }) {
  if (rows.length === 0) return null
  return (
    <Card title={title}>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="bg-slate-50">
            <tr>
              {columns.map((c) => (
                <th key={c.label} scope="col" className="px-3 py-2 text-left text-xs font-semibold tracking-wide whitespace-nowrap text-slate-600 uppercase">
                  {c.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {rows.map((r, i) => (
              <tr key={`${String(r.row)}-${String(i)}`}>
                {columns.map((c) => (
                  <td key={c.label} className={`px-3 py-2 align-top text-slate-800 ${c.wrap ? 'min-w-56 whitespace-normal' : 'whitespace-nowrap'}`}>
                    {c.render(r)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}

const cell = (key: string) => (r: CvRow) => {
  const value = r.cells[key] ?? ''
  return value === '' ? '—' : value
}

export function CvImportPanel() {
  const qc = useQueryClient()
  const toast = useToast()
  const navigate = useNavigate()
  const { options } = useOptions()
  const input = useRef<HTMLInputElement>(null)
  const [preview, setPreview] = useState<CvPreview | null>(null)
  const [fileName, setFileName] = useState('')
  const [decisions, setDecisions] = useState<Record<string, CvDecision>>({})

  const validate = useMutation({
    mutationFn: crewCvApi.validate,
    onMutate: () => {
      setPreview(null)
      setDecisions({})
    },
    onSuccess: (p) => {
      setPreview(p)
      setDecisions(initialDecisions(p.choices))
    },
  })
  const commit = useMutation({
    mutationFn: (p: CvPreview) => crewCvApi.commit(p.token, decisions),
    onSuccess: (r) => {
      void qc.invalidateQueries({ queryKey: crewKeys.all })
      void qc.invalidateQueries({ queryKey: ['grid', 'crews'] })
      void qc.invalidateQueries({ queryKey: OPTIONS_KEY })
      toast.success(`Imported ${r.full_name} from CV.`)
      void navigate(`/crews/${r.crew_id}`)
    },
  })

  const certificateTypes = { label: 'Match to a certificate type', items: options.certificate_types }
  const documentTypes = { label: 'Match to a document type', items: options.document_types }
  const groups: Record<CvChoiceKind, Group[]> = {
    certificate: [certificateTypes, documentTypes],
    document: [documentTypes],
    rank: [{ label: 'Match to a rank', items: options.ranks.map((r) => ({ id: r.id, name: `${r.name} (${r.short_code})` })) }],
    vessel_type: [{ label: 'Match to a vessel type', items: options.vessel_types }],
  }
  const names = new Map<string, string>()
  for (const list of [options.certificate_types, options.document_types, options.ranks, options.vessel_types]) {
    for (const it of list) names.set(it.id, it.name)
  }
  const describe = (d: CvDecision) => (d.action === 'match' ? (names.get(d.target_id ?? '') ?? 'Matched') : d.action === 'create' ? 'New type' : 'Skipped')
  const status = (r: CvRow): ReactNode => {
    const keys = r.choice_keys ?? []
    if (keys.some((k) => !(k in decisions))) return <Badge tone="yellow">Needs a choice</Badge>
    if (r.matched_as) return r.matched_as
    const chosen = keys.flatMap((k) => {
      const d = decisions[k]
      return d ? [describe(d)] : []
    })
    return chosen.length > 0 ? chosen.join(', ') : 'Ready'
  }

  const failure = validate.error ?? commit.error
  const blocked = preview ? preview.token === '' || preview.errors.length > 0 || preview.duplicate !== undefined : true
  const decided = preview ? preview.choices.every((c) => c.key in decisions) : false

  return (
    <div className="space-y-4">
      <Card title="Upload the CV">
        <p className="mb-3 text-sm text-slate-600">Use the RVOS CV layout saved as .xlsx. Nothing is saved until you confirm.</p>
        <div className="flex flex-wrap items-center gap-3">
          <input
            ref={input}
            type="file"
            accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
            aria-label="CV file (.xlsx)"
            className="sr-only"
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) {
                setFileName(file.name)
                validate.mutate(file)
              }
              e.target.value = ''
            }}
          />
          <Button variant="secondary" icon={<Icon name="upload" />} loading={validate.isPending} onClick={() => input.current?.click()}>
            {preview ? 'Choose another file' : 'Choose CV file'}
          </Button>
          {fileName ? <span className="min-w-0 text-sm break-all text-slate-700">{fileName}</span> : null}
        </div>
      </Card>

      {failure ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
          {fieldErrors(failure).file ?? errorMessage(failure)}
        </p>
      ) : null}

      {preview ? (
        <>
          <p className="text-sm font-medium text-navy-800">{summaryText(preview)}</p>

          {preview.duplicate ? (
            <div role="alert" className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-md bg-red-50 p-3 text-sm text-red-700">
              <p>{`${preview.duplicate.full_name} is already registered (${reasonText[preview.duplicate.reason]}).`}</p>
              <Link to={`/crews/${preview.duplicate.crew_id}`} className="font-medium underline">
                Open profile
              </Link>
            </div>
          ) : null}

          {preview.errors.length > 0 ? (
            <div role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
              <p className="mb-1 font-medium">Fix these cells in the file and upload it again:</p>
              <ul className="list-disc space-y-0.5 pl-5">
                {preview.errors.map((e, i) => (
                  <li key={`${e.section}-${String(e.row)}-${String(i)}`}>{`${sectionLabel[e.section]} row ${String(e.row)}: ${e.message}`}</li>
                ))}
              </ul>
            </div>
          ) : null}

          {preview.choices.length > 0 && !preview.duplicate ? (
            <Card title="Needs your choice">
              <div className="space-y-3">
                {preview.choices.map((c) => (
                  <ChoiceField
                    key={c.key}
                    choice={c}
                    value={encode(decisions[c.key])}
                    groups={groups[c.kind]}
                    onChange={(value) =>
                      setDecisions((d) => {
                        const rest = Object.fromEntries(Object.entries(d).filter(([k]) => k !== c.key))
                        const decision = decode(value)
                        return decision ? { ...rest, [c.key]: decision } : rest
                      })
                    }
                  />
                ))}
              </div>
            </Card>
          ) : null}

          <Card title="Personal data">
            <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
              {preview.personal.map((f) => (
                <div key={f.label} className="min-w-0">
                  <dt className="text-xs text-slate-500">{f.label}</dt>
                  <dd className="flex flex-wrap items-center gap-2 text-sm break-words text-slate-900">
                    <span>{f.value || '—'}</span>
                    {f.choice_key && !(f.choice_key in decisions) ? <Badge tone="yellow">Needs a choice</Badge> : null}
                  </dd>
                </div>
              ))}
            </dl>
          </Card>

          <CvTable
            title="Next of kin"
            rows={preview.next_of_kin}
            columns={[
              { label: 'Name', render: cell('name') },
              { label: 'Relationship', render: (r) => (r.cells.relationship_text ? `${r.cells.relationship ?? ''} (${r.cells.relationship_text})` : (r.cells.relationship ?? '—')) },
              { label: 'Phone', render: cell('phone') },
              { label: 'Address', render: cell('address'), wrap: true },
            ]}
          />
          <CvTable
            title="Documents"
            rows={preview.documents}
            columns={[
              { label: 'Document in CV', render: cell('type') },
              { label: 'In the system', render: status },
              { label: 'Number', render: cell('number') },
              { label: 'Issued', render: cell('issued') },
              { label: 'Expires', render: cell('expires') },
              { label: 'Place', render: cell('place') },
            ]}
          />
          <CvTable
            title="Certificates"
            rows={preview.certificates}
            columns={[
              { label: 'Certificate in CV', render: cell('text'), wrap: true },
              { label: 'In the system', render: status },
              { label: 'Issued', render: cell('issued') },
              { label: 'Expires', render: cell('expires') },
            ]}
          />
          <CvTable
            title="Sea service"
            rows={preview.sea_service}
            columns={[
              { label: 'Vessel', render: cell('vessel') },
              { label: 'Rank', render: cell('rank') },
              { label: 'Vessel type', render: cell('vessel_type') },
              { label: 'GRT / HP', render: cell('grt_hp') },
              { label: 'Company', render: cell('company') },
              { label: 'From', render: cell('from') },
              { label: 'To', render: cell('to') },
              { label: 'Flag', render: cell('flag') },
              { label: 'Status', render: status },
            ]}
          />

          {preview.not_imported.length > 0 ? (
            <Card title="Not imported">
              <ul className="list-disc space-y-0.5 pl-5 text-sm text-slate-700">
                {preview.not_imported.map((n) => (
                  <li key={n}>{n}</li>
                ))}
              </ul>
            </Card>
          ) : null}
          {preview.warnings.length > 0 ? (
            <Card title="Warnings">
              <ul className="list-disc space-y-0.5 pl-5 text-sm text-yellow-800">
                {preview.warnings.map((w) => (
                  <li key={w}>{w}</li>
                ))}
              </ul>
            </Card>
          ) : null}

          {preview.duplicate ? null : (
            <div className="flex justify-end gap-2 max-sm:*:flex-1">
              <Button
                variant="secondary"
                disabled={commit.isPending}
                onClick={() => {
                  setPreview(null)
                  setFileName('')
                }}
              >
                Discard
              </Button>
              <Button disabled={blocked || !decided || commit.isPending} loading={commit.isPending} onClick={() => commit.mutate(preview)}>
                Confirm import
              </Button>
            </div>
          )}
        </>
      ) : null}
    </div>
  )
}
