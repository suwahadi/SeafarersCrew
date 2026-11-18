import { useState } from 'react'
import type { PayrollFacets } from '@/api/payroll'
import { MultiSelect } from '@/components/forms/MultiSelect'
import { Button } from '@/components/ui/Button'
import { EMPTY_FILTERS, type PayrollFilters } from './filters'

interface Props {
  value: PayrollFilters
  facets: PayrollFacets
  onApply: (filters: PayrollFilters) => void
}

const options = (values: string[]) => values.map((v) => ({ value: v, label: v }))

export function PayrollFilterPanel({ value, facets, onApply }: Props) {
  const [draft, setDraft] = useState<PayrollFilters>(value)
  const set = <K extends keyof PayrollFilters>(key: K, v: PayrollFilters[K]) => setDraft((d) => ({ ...d, [key]: v }))

  return (
    <form
      aria-label="Payroll filters"
      onSubmit={(e) => {
        e.preventDefault()
        onApply(draft)
      }}
      className="mb-4 rounded-lg bg-white p-3 shadow-sm ring-1 ring-slate-200 sm:p-4"
    >
      <div className="flex flex-wrap items-end gap-x-4 gap-y-3">
        <div className="min-w-44 flex-1">
          <MultiSelect label="Bank" options={options(facets.banks)} value={draft.bank} onChange={(v) => set('bank', v)} />
        </div>
        <div className="min-w-44 flex-1">
          <MultiSelect label="Vessel" options={options(facets.vessels)} value={draft.vessel} onChange={(v) => set('vessel', v)} />
        </div>
        <div className="min-w-36 flex-1">
          <MultiSelect label="Rank" options={options(facets.ranks)} value={draft.rank} onChange={(v) => set('rank', v)} />
        </div>
        <fieldset className="min-w-0">
          <legend className="mb-1 text-sm font-medium text-slate-700">Show only</legend>
          <div className="flex flex-wrap gap-x-4">
            <label className="flex min-h-10 items-center gap-2 text-sm whitespace-nowrap text-slate-700">
              <input type="checkbox" checked={draft.has_remark} onChange={(e) => set('has_remark', e.target.checked)} className="h-4 w-4 rounded border-slate-300" />
              With a remark
            </label>
            <label className="flex min-h-10 items-center gap-2 text-sm whitespace-nowrap text-slate-700">
              <input type="checkbox" checked={draft.no_usd} onChange={(e) => set('no_usd', e.target.checked)} className="h-4 w-4 rounded border-slate-300" />
              Without USD
            </label>
          </div>
        </fieldset>
        <div className="flex basis-full justify-end gap-2 sm:ml-auto sm:basis-auto">
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
      </div>
    </form>
  )
}
