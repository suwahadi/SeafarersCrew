import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { errorMessage } from '@/api/client'
import { complianceApi, COMPLIANCE_KEY, type ComplianceMatrix, type Rank } from '@/api/masters'
import { Button } from '@/components/ui/Button'
import { Badge, PageHeader } from '@/components/ui/Layout'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { departments } from './config'
import { cellKey, cellsToSet, columnsOf, sameSet, setToCells } from './matrix'

export default function ComplianceMatrixPage() {
  const matrix = useQuery({ queryKey: COMPLIANCE_KEY, queryFn: ({ signal }) => complianceApi.get(signal) })
  if (matrix.isPending) {
    return (
      <div className="flex justify-center p-12 text-navy-800">
        <Spinner size="lg" />
      </div>
    )
  }
  if (matrix.isError) return <p role="alert" className="text-sm text-red-700">{errorMessage(matrix.error)}</p>
  return <MatrixEditor matrix={matrix.data} />
}

export function MatrixEditor({ matrix }: { matrix: ComplianceMatrix }) {
  const qc = useQueryClient()
  const toast = useToast()
  const initial = useMemo(() => cellsToSet(matrix.cells), [matrix.cells])
  const [checked, setChecked] = useState<Set<string>>(() => new Set(initial))
  const columns = useMemo(() => columnsOf(matrix), [matrix])
  const dirty = !sameSet(checked, initial)

  const save = useMutation({
    mutationFn: () => complianceApi.replace(setToCells(checked)),
    onSuccess: (data) => {
      qc.setQueryData(COMPLIANCE_KEY, data)
      toast.success('Compliance matrix saved.')
    },
    onError: (err) => toast.error(errorMessage(err)),
  })

  const update = (keys: string[], on: boolean) =>
    setChecked((cur) => {
      const next = new Set(cur)
      for (const k of keys) {
        if (on) next.add(k)
        else next.delete(k)
      }
      return next
    })

  const rowKeys = (r: Rank) => columns.map((c) => cellKey(r.id, c.kind, c.id))
  const colKeys = (c: (typeof columns)[number]) => matrix.ranks.map((r) => cellKey(r.id, c.kind, c.id))
  const allOn = (keys: string[]) => keys.length > 0 && keys.every((k) => checked.has(k))

  const grouped = departments.map((d) => ({ dept: d, ranks: matrix.ranks.filter((r) => r.department === d.value) })).filter((g) => g.ranks.length > 0)

  return (
    <div>
      <PageHeader
        title="Compliance Matrix"
        description="Mandatory documents and certificates per rank. A crew member is compliant when every checked item exists and is valid."
        actions={
          <>
            {dirty ? <Badge tone="orange">Unsaved changes</Badge> : null}
            <Button variant="secondary" disabled={!dirty || save.isPending} onClick={() => setChecked(new Set(initial))}>
              Discard
            </Button>
            <Button onClick={() => save.mutate()} loading={save.isPending} disabled={!dirty}>
              Save matrix
            </Button>
          </>
        }
      />
      <div className="max-h-[75vh] overflow-auto rounded-lg bg-white shadow-sm ring-1 ring-slate-200">
        <table className="border-separate border-spacing-0 text-xs">
          <thead>
            <tr>
              <th scope="col" className="sticky top-0 left-0 z-30 min-w-48 border-b border-slate-200 bg-slate-50 px-3 py-2 text-left font-semibold text-slate-600">
                Rank
              </th>
              {columns.map((c) => (
                <th key={`${c.kind}-${c.id}`} scope="col" className="sticky top-0 z-20 border-b border-l border-slate-200 bg-slate-50 px-2 py-2 align-bottom font-medium text-slate-700">
                  <div className="flex min-w-20 flex-col items-center gap-1">
                    <span className={`rounded px-1 text-[10px] uppercase ${c.kind === 'document' ? 'bg-blue-50 text-blue-700' : 'bg-green-50 text-green-700'}`}>
                      {c.kind === 'document' ? 'Doc' : 'Cert'}
                    </span>
                    <span className="max-w-28 text-center leading-tight">{c.name}</span>
                    <input
                      type="checkbox"
                      aria-label={`Require ${c.name} for all ranks`}
                      checked={allOn(colKeys(c))}
                      onChange={(e) => update(colKeys(c), e.target.checked)}
                    />
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          {grouped.map((g) => (
            <tbody key={g.dept.value}>
              <tr>
                <th colSpan={columns.length + 1} scope="rowgroup" className="sticky left-0 bg-navy-50 px-3 py-1 text-left font-semibold text-navy-800">
                  {g.dept.label}
                </th>
              </tr>
              {g.ranks.map((r) => (
                <tr key={r.id} className="hover:bg-slate-50">
                  <th scope="row" className="sticky left-0 z-10 border-b border-slate-100 bg-white px-3 py-1.5 text-left font-medium whitespace-nowrap text-slate-800">
                    <label className="flex items-center gap-2">
                      <input type="checkbox" aria-label={`Require all items for ${r.name}`} checked={allOn(rowKeys(r))} onChange={(e) => update(rowKeys(r), e.target.checked)} />
                      {r.name} <span className="text-slate-400">{r.short_code}</span>
                    </label>
                  </th>
                  {columns.map((c) => {
                    const key = cellKey(r.id, c.kind, c.id)
                    return (
                      <td key={key} className="border-b border-l border-slate-100 text-center">
                        <input type="checkbox" aria-label={`${r.name}: ${c.name}`} checked={checked.has(key)} onChange={(e) => update([key], e.target.checked)} className="h-4 w-4" />
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          ))}
        </table>
      </div>
    </div>
  )
}
