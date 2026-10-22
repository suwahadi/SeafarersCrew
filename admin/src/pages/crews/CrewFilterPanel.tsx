import { useState } from 'react'
import { DEPARTMENTS } from '@/api/options'
import { MultiSelect } from '@/components/forms/MultiSelect'
import { Button } from '@/components/ui/Button'
import { SelectField, TextField } from '@/components/ui/Field'
import { useOptions } from '@/hooks/useOptions'
import { EMPTY_FILTERS, type CrewFilters } from './filters'

interface Props {
  value: CrewFilters
  onApply: (filters: CrewFilters) => void
}

const COMPLIANCE = [
  { value: 'compliant', label: 'Compliant' },
  { value: 'not_compliant', label: 'Not compliant' },
]

const VALIDITY = [
  { value: 'valid', label: 'All valid' },
  { value: 'expiring', label: 'Expiring within N days' },
  { value: 'expired', label: 'Has expired items' },
]

const GROUPS = [
  { value: 'principal', label: 'Principal' },
  { value: 'company', label: 'Employer company' },
]

export function CrewFilterPanel({ value, onApply }: Props) {
  const { options } = useOptions()
  const [draft, setDraft] = useState<CrewFilters>(value)
  const set = <K extends keyof CrewFilters>(key: K, v: CrewFilters[K]) => setDraft((d) => ({ ...d, [key]: v }))

  const statusOptions = options.crew_statuses.filter((s) => s.code).map((s) => ({ value: s.code ?? '', label: s.name }))
  const rankOptions = options.ranks.map((r) => ({ value: r.id, label: `${r.name} (${r.short_code})` }))
  const principals = options.companies.filter((c) => c.type !== 'company').map((c) => ({ value: c.id, label: c.name }))

  return (
    <form
      aria-label="Crew filters"
      onSubmit={(e) => {
        e.preventDefault()
        onApply(draft)
      }}
      className="mb-4 rounded-lg bg-white p-3 shadow-sm ring-1 ring-slate-200 sm:p-4"
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MultiSelect label="Status" options={statusOptions} value={draft.status} onChange={(v) => set('status', v)} />
        <MultiSelect label="Rank" options={rankOptions} value={draft.rank_id} onChange={(v) => set('rank_id', v)} />
        <SelectField label="Department" placeholder="Any" options={[...DEPARTMENTS]} value={draft.department} onChange={(e) => set('department', e.target.value)} />
        <SelectField
          label="Vessel type"
          placeholder="Any"
          options={options.vessel_types.map((t) => ({ value: t.id, label: t.name }))}
          value={draft.vessel_type_id}
          onChange={(e) => set('vessel_type_id', e.target.value)}
        />
        <SelectField
          label="Current vessel"
          placeholder="Any"
          options={options.vessels.map((v) => ({ value: v.id, label: v.name }))}
          value={draft.vessel_id}
          onChange={(e) => set('vessel_id', e.target.value)}
        />
        <SelectField label="Principal" placeholder="Any" options={principals} value={draft.principal_id} onChange={(e) => set('principal_id', e.target.value)} />
        <div className="grid grid-cols-2 gap-2">
          <TextField label="Age from" type="number" min={0} max={100} inputMode="numeric" value={draft.age_min} onChange={(e) => set('age_min', e.target.value)} />
          <TextField label="Age to" type="number" min={0} max={100} inputMode="numeric" value={draft.age_max} onChange={(e) => set('age_max', e.target.value)} />
        </div>
        <TextField label="Nationality" value={draft.nationality} onChange={(e) => set('nationality', e.target.value)} placeholder="e.g. Indonesian" />
        <SelectField label="Compliance" placeholder="Any" options={COMPLIANCE} value={draft.compliance} onChange={(e) => set('compliance', e.target.value)} />
        <div className="grid grid-cols-[2fr_1fr] gap-2">
          <SelectField label="Document validity" placeholder="Any" options={VALIDITY} value={draft.doc_validity} onChange={(e) => set('doc_validity', e.target.value)} />
          <TextField
            label="Days"
            type="number"
            min={1}
            max={3650}
            placeholder="90"
            disabled={draft.doc_validity === ''}
            value={draft.doc_days}
            onChange={(e) => set('doc_days', e.target.value)}
          />
        </div>
        <div className="grid grid-cols-[2fr_1fr_1fr] gap-2 sm:col-span-2 lg:col-span-2">
          <SelectField label="Minimum sea service in rank" placeholder="Any rank" options={rankOptions} value={draft.sea_rank_id} onChange={(e) => set('sea_rank_id', e.target.value)} />
          <TextField label="Years" type="number" min={0} disabled={draft.sea_rank_id === ''} value={draft.sea_min_years} onChange={(e) => set('sea_min_years', e.target.value)} />
          <TextField label="Months" type="number" min={0} max={11} disabled={draft.sea_rank_id === ''} value={draft.sea_min_months} onChange={(e) => set('sea_min_months', e.target.value)} />
        </div>
        <SelectField label="Group by" placeholder="No grouping" options={GROUPS} value={draft.group_by} onChange={(e) => set('group_by', e.target.value)} />
      </div>
      <div className="mt-4 grid grid-cols-2 gap-2 sm:flex sm:justify-end">
        <Button
          variant="ghost"
          onClick={() => {
            setDraft(EMPTY_FILTERS)
            onApply(EMPTY_FILTERS)
          }}
        >
          Reset
        </Button>
        <Button type="submit">Apply filters</Button>
      </div>
    </form>
  )
}
