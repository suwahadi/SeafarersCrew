import { keepPreviousData, useQueries, useQuery } from '@tanstack/react-query'
import { useRef, useState, type DragEvent } from 'react'
import { Link } from 'react-router'
import { applicantsApi, type Applicant, type Stage } from '@/api/applicants'
import { errorMessage, type Query } from '@/api/client'
import { Spinner } from '@/components/ui/Spinner'
import { formatDateTime } from '@/lib/format'
import { STAGES, STAGE_LABELS, emptyCounts, groupByStage, moveTargets } from '@/lib/pipeline'
import { StageMoveDialog } from './StageMoveDialog'
import { useStageMove } from './useStageMove'

const COLUMN_SIZE = 50

export function ApplicantKanban({ filters }: { filters: Query }) {
  const move = useStageMove()
  const dragged = useRef<{ id: string; name: string; from: Stage } | null>(null)
  const [over, setOver] = useState<Stage | null>(null)

  const counts = useQuery({
    queryKey: ['applicants', 'stage-counts', filters],
    queryFn: ({ signal }) => applicantsApi.stageCounts(filters, signal),
    placeholderData: keepPreviousData,
  })

  const columns = useQueries({
    queries: STAGES.map((stage) => ({
      queryKey: ['applicants', 'kanban', stage, filters],
      queryFn: ({ signal }: { signal: AbortSignal }) => applicantsApi.list({ ...filters, stage, per_page: COLUMN_SIZE }, signal),
      placeholderData: keepPreviousData,
    })),
  })

  const totals = counts.data ?? emptyCounts()
  const error = counts.error ?? columns.find((c) => c.isError)?.error

  const onDrop = (e: DragEvent, to: Stage) => {
    e.preventDefault()
    setOver(null)
    const d = dragged.current
    dragged.current = null
    if (d && d.from !== to) move.start({ ...d, to })
  }

  return (
    <>
      {error ? (
        <p role="alert" className="mb-3 text-sm text-red-700">
          {errorMessage(error)}
        </p>
      ) : null}
      <div className="grid auto-cols-[82%] grid-flow-col gap-3 overflow-x-auto pb-2 sm:auto-cols-[45%] lg:grid-flow-row lg:grid-cols-[repeat(5,minmax(13rem,1fr))]" role="group" aria-label="Applicant pipeline">
        {STAGES.map((stage, i) => {
          const result = columns[i]
          const items = groupByStage(result?.data?.data ?? [])[stage]
          const total = result?.data?.meta.total ?? totals[stage]
          return (
            <section
              key={stage}
              aria-labelledby={`col-${stage}`}
              onDragOver={(e) => {
                if (dragged.current) {
                  e.preventDefault()
                  setOver(stage)
                }
              }}
              onDragLeave={() => setOver((o) => (o === stage ? null : o))}
              onDrop={(e) => onDrop(e, stage)}
              className={`flex min-h-48 min-w-0 flex-col rounded-lg p-2 ring-1 transition-colors duration-150 ease-out ${over === stage ? 'bg-navy-50 ring-navy-400' : 'bg-slate-50 ring-slate-200'}`}
            >
              <header className="mb-2 flex items-center justify-between px-1">
                <h2 id={`col-${stage}`} className="text-sm font-semibold text-slate-800">
                  {STAGE_LABELS[stage]}
                </h2>
                <span className="rounded-full bg-white px-2 py-0.5 text-xs font-medium text-slate-700 ring-1 ring-slate-200" aria-label={`${totals[stage]} applicants`}>
                  {totals[stage]}
                </span>
              </header>
              <ul className="relative flex flex-1 flex-col gap-2">
                {result?.isPending ? (
                  <li className="flex justify-center p-4 text-navy-800">
                    <Spinner label={`Loading ${STAGE_LABELS[stage]} applicants`} />
                  </li>
                ) : items.length === 0 ? (
                  <li className="p-3 text-center text-xs text-slate-500">No applicants</li>
                ) : (
                  items.map((a) => (
                    <KanbanCard
                      key={a.id}
                      applicant={a}
                      onDragStart={() => {
                        dragged.current = { id: a.id, name: a.full_name, from: a.stage }
                      }}
                      onDragEnd={() => {
                        dragged.current = null
                        setOver(null)
                      }}
                      onMove={(to) => move.start({ id: a.id, name: a.full_name, from: a.stage, to })}
                    />
                  ))
                )}
              </ul>
              {total > items.length ? <p className="mt-2 px-1 text-xs text-slate-500">Showing {items.length} of {total}. Narrow the filters to see the rest.</p> : null}
            </section>
          )
        })}
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
    </>
  )
}

interface KanbanCardProps {
  applicant: Applicant
  onDragStart: () => void
  onDragEnd: () => void
  onMove: (to: Stage) => void
}

function KanbanCard({ applicant: a, onDragStart, onDragEnd, onMove }: KanbanCardProps) {
  return (
    <li
      draggable
      onDragStart={(e) => {
        e.dataTransfer.setData('text/plain', a.id)
        e.dataTransfer.effectAllowed = 'move'
        onDragStart()
      }}
      onDragEnd={onDragEnd}
      className="cursor-grab rounded-md bg-white p-3 text-sm shadow-sm ring-1 ring-slate-200 active:cursor-grabbing"
    >
      <Link to={`/applicants/${a.id}`} className="font-medium text-navy-700 hover:underline">
        {a.full_name}
      </Link>
      <p className="text-xs text-slate-500">{a.id}</p>
      <p className="mt-1 text-xs text-slate-700">{a.vacancy.title}</p>
      <p className="text-xs text-slate-600">
        {a.rank_applied?.name ?? 'No rank'}
        {a.years_experience !== null ? ` · ${a.years_experience} yr${a.years_experience === 1 ? '' : 's'}` : ''}
      </p>
      <p className="text-xs text-slate-500">{formatDateTime(a.created_at)}</p>
      <label className="relative mt-2 block">
        <span className="sr-only">Move {a.full_name} to another stage</span>
        <select
          value=""
          onChange={(e) => {
            const to = moveTargets(a.stage).find((s) => s === e.target.value)
            if (to) onMove(to)
          }}
          className="h-9 w-full rounded-md border-0 bg-white px-2 text-xs ring-1 ring-inset ring-slate-300 focus:ring-2 focus:ring-navy-500"
        >
          <option value="">Move to…</option>
          {moveTargets(a.stage).map((s) => (
            <option key={s} value={s}>
              {STAGE_LABELS[s]}
            </option>
          ))}
        </select>
      </label>
    </li>
  )
}
