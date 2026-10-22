import type { Timeline, TimelineBar, TimelinePlan } from '@/api/contracts'
import { formatDate } from '@/lib/format'
import { barPosition, markerPosition, monthTicks, type BarState } from '@/lib/timeline'

const STATE_CLASS: Record<BarState, string> = {
  normal: 'bg-navy-500 hover:bg-navy-700 text-white',
  due_soon: 'bg-orange-600 hover:bg-orange-700 text-white',
  overdue: 'bg-red-600 hover:bg-red-700 text-white',
}

export const STATE_LABEL: Record<BarState, string> = { normal: 'On schedule', due_soon: 'Sign-off due soon', overdue: 'Sign-off overdue' }

interface Props {
  timeline: Timeline
  onSelectBar: (bar: TimelineBar, vesselName: string) => void
  onSelectPlan: (plan: TimelinePlan, bar: TimelineBar, vesselName: string) => void
}

export function RotationTimeline({ timeline, onSelectBar, onSelectPlan }: Props) {
  const { from, to, today, lanes } = timeline
  const ticks = monthTicks(from, to)
  const todayLeft = markerPosition(today, from, to)

  return (
    <div className="overflow-x-auto rounded-lg bg-white ring-1 ring-slate-200">
      <div className="min-w-[56rem]">
        <div className="grid grid-cols-[11rem_1fr] border-b border-slate-200 bg-slate-50 text-xs text-slate-600">
          <div className="px-3 py-2 font-semibold uppercase">Vessel</div>
          <div className="relative h-8" aria-hidden>
            {ticks.map((t) => (
              <span key={t.label} className="absolute top-2 border-l border-slate-300 pl-1" style={{ left: `${t.left}%` }}>
                {t.label}
              </span>
            ))}
          </div>
        </div>
        {lanes.length === 0 ? <p className="p-6 text-center text-sm text-slate-500">No vessels match this filter.</p> : null}
        {lanes.map((lane) => (
          <div key={lane.vessel.id} className="grid grid-cols-[11rem_1fr] border-b border-slate-100 last:border-0">
            <div className="px-3 py-2">
              <p className="text-sm font-medium text-slate-800">{lane.vessel.name}</p>
              <p className="text-xs text-slate-500">{lane.vessel_type || '—'}</p>
            </div>
            <div className="relative py-1">
              {ticks.map((t) => (
                <span key={t.label} aria-hidden className="absolute inset-y-0 border-l border-slate-100" style={{ left: `${t.left}%` }} />
              ))}
              {todayLeft !== null ? <span aria-hidden className="absolute inset-y-0 border-l-2 border-navy-800" style={{ left: `${todayLeft}%` }} /> : null}
              {lane.bars.length === 0 ? <p className="px-2 py-1 text-xs text-slate-400">No contracts in this range.</p> : null}
              {lane.bars.map((bar) => {
                const pos = barPosition(bar.sign_on_date, bar.planned_sign_off_date, from, to)
                return (
                  <div key={bar.contract_id} className="relative h-8" data-testid={`row-${bar.contract_id}`}>
                    {pos.visible ? (
                      <button
                        type="button"
                        onClick={() => onSelectBar(bar, lane.vessel.name)}
                        aria-label={`${bar.crew.name}, ${bar.rank_name}, ${formatDate(bar.sign_on_date)} to ${formatDate(bar.planned_sign_off_date)}, ${STATE_LABEL[bar.state]}`}
                        data-state={bar.state}
                        className={`absolute top-1 h-6 overflow-hidden rounded px-2 text-left text-xs leading-6 whitespace-nowrap transition-colors duration-150 ${STATE_CLASS[bar.state]} ${pos.clippedStart ? 'rounded-l-none' : ''} ${pos.clippedEnd ? 'rounded-r-none' : ''}`}
                        style={{ left: `${pos.left}%`, width: `${pos.width}%` }}
                      >
                        {bar.crew.name} · {bar.rank_name}
                      </button>
                    ) : null}
                    {bar.relief_plans.map((plan) => {
                      const left = markerPosition(plan.planned_relief_date, from, to)
                      if (left === null) return null
                      return (
                        <button
                          key={plan.id}
                          type="button"
                          onClick={() => onSelectPlan(plan, bar, lane.vessel.name)}
                          aria-label={`Relief by ${plan.reliever.name} on ${formatDate(plan.planned_relief_date)}, ${plan.status}`}
                          className="absolute top-0 z-20 -ml-2 h-4 w-4 rotate-45 rounded-sm bg-white ring-2 ring-navy-800 hover:bg-navy-100"
                          style={{ left: `${left}%` }}
                        />
                      )
                    })}
                  </div>
                )
              })}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export function TimelineLegend() {
  return (
    <ul className="flex flex-wrap gap-4 text-xs text-slate-600" aria-label="Legend">
      <li className="flex items-center gap-1">
        <span className="h-3 w-5 rounded bg-navy-500" /> On schedule
      </li>
      <li className="flex items-center gap-1">
        <span className="h-3 w-5 rounded bg-orange-600" /> Due within 30 days
      </li>
      <li className="flex items-center gap-1">
        <span className="h-3 w-5 rounded bg-red-600" /> Overdue
      </li>
      <li className="flex items-center gap-1">
        <span className="h-3 w-3 rotate-45 rounded-sm bg-white ring-2 ring-navy-800" /> Relief plan
      </li>
      <li className="flex items-center gap-1">
        <span className="h-4 border-l-2 border-navy-800" /> Today
      </li>
    </ul>
  )
}
