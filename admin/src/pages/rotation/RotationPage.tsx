import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router'
import { errorMessage } from '@/api/client'
import { rotationApi, type ReliefPlan, type TimelineBar } from '@/api/contracts'
import { Button } from '@/components/ui/Button'
import { TextField, inputClass } from '@/components/ui/Field'
import { Badge, DescriptionList, PageHeader } from '@/components/ui/Layout'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Tabs } from '@/components/ui/Tabs'
import { useOptions } from '@/hooks/useOptions'
import { formatDate, todayISO } from '@/lib/format'
import { addDays } from '@/lib/timeline'
import { ReliefPlanDialog, type ReliefTarget } from './ReliefPlanDialog'
import { ReliefPlanList } from './ReliefPlanList'
import { RotationTimeline, STATE_LABEL, TimelineLegend } from './RotationTimeline'

interface Selected {
  bar: TimelineBar
  vesselName: string
}

type Editing = { target: ReliefTarget; plan?: ReliefPlan } | null

const TABS = [
  { id: 'timeline', label: 'Timeline' },
  { id: 'plans', label: 'Relief plans' },
]

export default function RotationPage() {
  const { options } = useOptions()
  const today = todayISO()
  const [tab, setTab] = useState('timeline')
  const [from, setFrom] = useState(addDays(today, -30))
  const [to, setTo] = useState(addDays(today, 180))
  const [vesselId, setVesselId] = useState('')
  const [selected, setSelected] = useState<Selected | null>(null)
  const [editing, setEditing] = useState<Editing>(null)
  const [planToOpen, setPlanToOpen] = useState<{ id: string; target: ReliefTarget } | null>(null)

  const rangeValid = from !== '' && to !== '' && from <= to
  const timeline = useQuery({
    queryKey: ['rotation', 'timeline', from, to, vesselId],
    queryFn: () => rotationApi.timeline({ from, to, vessel_id: vesselId }),
    enabled: rangeValid,
  })

  const targetOf = (bar: TimelineBar, vesselName: string): ReliefTarget => ({
    contractId: bar.contract_id,
    label: `${bar.crew.name} (${bar.rank_name}) on ${vesselName}`,
    defaultDate: bar.planned_sign_off_date,
  })

  return (
    <>
      <PageHeader title="Rotation planning" description="Sign-on and planned sign-off per vessel, with relief plans." />
      <Tabs tabs={TABS} active={tab} onChange={setTab} />
      <div className="mt-4">
        {tab === 'plans' ? (
          <ReliefPlanList onOpen={(plan) => setEditing({ target: { contractId: plan.contract_id, label: `${plan.relieved.name} (${plan.rank.name}) on ${plan.vessel.name}`, defaultDate: plan.planned_relief_date }, plan })} />
        ) : (
          <div className="space-y-3">
            <div className="flex flex-wrap items-end gap-3">
              <div className="w-40">
                <TextField label="From" type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
              </div>
              <div className="w-40">
                <TextField label="To" type="date" value={to} onChange={(e) => setTo(e.target.value)} error={from !== '' && to !== '' && from > to ? 'Must be after From.' : undefined} />
              </div>
              <div>
                <label htmlFor="rotation-vessel" className="mb-1 block text-sm font-medium text-slate-700">
                  Vessel
                </label>
                <select id="rotation-vessel" className={`h-9 w-56 ${inputClass}`} value={vesselId} onChange={(e) => setVesselId(e.target.value)}>
                  <option value="">All vessels</option>
                  {options.vessels.map((v) => (
                    <option key={v.id} value={v.id}>
                      {v.name}
                    </option>
                  ))}
                </select>
              </div>
              <Button
                variant="secondary"
                onClick={() => {
                  setFrom(addDays(today, -30))
                  setTo(addDays(today, 180))
                }}
              >
                Reset range
              </Button>
              {timeline.isFetching ? <Spinner size="sm" label="Updating timeline" /> : null}
            </div>
            <TimelineLegend />
            {timeline.isError ? <p role="alert" className="text-sm text-red-700">{errorMessage(timeline.error)}</p> : null}
            {timeline.isPending && rangeValid ? (
              <div className="flex justify-center p-12 text-navy-800">
                <Spinner size="lg" label="Loading timeline" />
              </div>
            ) : null}
            {timeline.data ? (
              <RotationTimeline
                timeline={timeline.data}
                onSelectBar={(bar, vesselName) => setSelected({ bar, vesselName })}
                onSelectPlan={(plan, bar, vesselName) => setPlanToOpen({ id: plan.id, target: targetOf(bar, vesselName) })}
              />
            ) : null}
          </div>
        )}
      </div>

      <Modal
        open={selected !== null}
        title={selected ? selected.bar.crew.name : ''}
        onClose={() => setSelected(null)}
        footer={
          selected ? (
            <>
              <Link to={`/contracts/${selected.bar.contract_id}`} className="mr-auto self-center text-sm text-navy-700 hover:underline">
                Open contract {selected.bar.contract_id}
              </Link>
              <Button variant="secondary" onClick={() => setSelected(null)}>
                Close
              </Button>
              <Button
                onClick={() => {
                  setEditing({ target: targetOf(selected.bar, selected.vesselName) })
                  setSelected(null)
                }}
              >
                Plan relief
              </Button>
            </>
          ) : undefined
        }
      >
        {selected ? (
          <div className="space-y-4">
            <DescriptionList
              items={[
                { label: 'Vessel', value: selected.vesselName },
                { label: 'Rank', value: selected.bar.rank_name },
                { label: 'Sign-on', value: formatDate(selected.bar.sign_on_date) },
                { label: 'Planned sign-off', value: formatDate(selected.bar.planned_sign_off_date) },
                {
                  label: 'Schedule',
                  value: <Badge tone={selected.bar.state === 'overdue' ? 'red' : selected.bar.state === 'due_soon' ? 'orange' : 'green'}>{STATE_LABEL[selected.bar.state]}</Badge>,
                },
              ]}
            />
            <div>
              <h3 className="mb-1 text-sm font-medium text-slate-700">Relief plans</h3>
              {selected.bar.relief_plans.length === 0 ? <p className="text-sm text-slate-500">No relief planned yet.</p> : null}
              <ul className="text-sm">
                {selected.bar.relief_plans.map((p) => (
                  <li key={p.id}>
                    {p.reliever.name} on {formatDate(p.planned_relief_date)} ({p.status})
                  </li>
                ))}
              </ul>
            </div>
          </div>
        ) : null}
      </Modal>

      {editing ? <ReliefPlanDialog target={editing.target} plan={editing.plan} onClose={() => setEditing(null)} /> : null}
      {planToOpen ? <PlanLoader id={planToOpen.id} target={planToOpen.target} onClose={() => setPlanToOpen(null)} /> : null}
    </>
  )
}

function PlanLoader({ id, target, onClose }: { id: string; target: ReliefTarget; onClose: () => void }) {
  const plan = useQuery({ queryKey: ['rotation', 'plan', id], queryFn: () => rotationApi.get(id) })
  if (plan.isError) {
    return (
      <Modal open title="Relief plan" onClose={onClose} footer={<Button onClick={onClose}>Close</Button>}>
        <p role="alert" className="text-sm text-red-700">{errorMessage(plan.error)}</p>
      </Modal>
    )
  }
  if (!plan.data) return null
  return <ReliefPlanDialog target={target} plan={plan.data} onClose={onClose} />
}
