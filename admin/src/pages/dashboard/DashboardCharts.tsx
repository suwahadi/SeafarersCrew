import { Bar, BarChart, CartesianGrid, Legend, Line, LineChart, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { CountByKey, DashboardCharts as Charts, MonthlyMovement } from '@/api/dashboard'
import { Card, EmptyState } from '@/components/ui/Layout'

const DEPARTMENT_COLORS: Record<string, string> = { deck: '#0B2545', engine: '#13A89E', catering_other: '#D97706' }
const DEPARTMENT_LABELS: Record<string, string> = { deck: 'Deck', engine: 'Engine', catering_other: 'Catering/Other' }
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

export function monthLabel(month: string): string {
  const [y, m] = month.split('-')
  return `${MONTHS[Number(m) - 1] ?? m} ${y?.slice(2) ?? ''}`
}

function ChartFrame({ title, height, empty, children }: { title: string; height: number; empty: boolean; children: React.ReactElement }) {
  return (
    <Card title={title}>
      {empty ? (
        <EmptyState title="No data yet" />
      ) : (
        <div style={{ height }} role="img" aria-label={title}>
          <ResponsiveContainer width="100%" height="100%">
            {children}
          </ResponsiveContainer>
        </div>
      )}
    </Card>
  )
}

export function StatusDonut({ data }: { data: CountByKey[] }) {
  const total = data.reduce((s, d) => s + d.count, 0)
  const slices = data.map((d) => ({ ...d, fill: d.color ?? '#64748B' }))
  return (
    <ChartFrame title="Crew by status" height={260} empty={total === 0}>
      <PieChart>
        <Pie data={slices} dataKey="count" nameKey="label" innerRadius={60} outerRadius={95} paddingAngle={1} isAnimationActive={false} />
        <Tooltip />
        <Legend height={36} />
      </PieChart>
    </ChartFrame>
  )
}

export function rankRows(data: CountByKey[]): Record<string, string | number>[] {
  return data
    .filter((d) => d.count > 0)
    .map((d) => ({ label: d.label, [d.group ?? 'catering_other']: d.count }))
}

export function RankBars({ data }: { data: CountByKey[] }) {
  const rows = rankRows(data)
  return (
    <ChartFrame title="Crew by rank" height={Math.max(220, rows.length * 26)} empty={rows.length === 0}>
      <BarChart data={rows} layout="vertical" margin={{ left: 24, right: 16 }}>
        <CartesianGrid strokeDasharray="3 3" horizontal={false} />
        <XAxis type="number" allowDecimals={false} />
        <YAxis type="category" dataKey="label" width={110} tick={{ fontSize: 12 }} />
        <Tooltip />
        <Legend />
        {Object.entries(DEPARTMENT_LABELS).map(([key, name]) => (
          <Bar key={key} dataKey={key} name={name} stackId="rank" fill={DEPARTMENT_COLORS[key]} isAnimationActive={false} />
        ))}
      </BarChart>
    </ChartFrame>
  )
}

export function PrincipalBars({ data }: { data: CountByKey[] }) {
  return (
    <ChartFrame title="Crew on board by principal" height={260} empty={data.length === 0}>
      <BarChart data={data} margin={{ left: 0, right: 16 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} />
        <XAxis dataKey="label" tick={{ fontSize: 12 }} interval={0} />
        <YAxis allowDecimals={false} />
        <Tooltip />
        <Bar dataKey="count" name="On board" fill="#0B2545" isAnimationActive={false} />
      </BarChart>
    </ChartFrame>
  )
}

export function MovementLines({ data }: { data: MonthlyMovement[] }) {
  const rows = data.map((d) => ({ ...d, label: monthLabel(d.month) }))
  return (
    <ChartFrame title="Sign-ons and sign-offs, last 12 months" height={260} empty={rows.length === 0}>
      <LineChart data={rows} margin={{ left: 0, right: 16 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} />
        <XAxis dataKey="label" tick={{ fontSize: 12 }} />
        <YAxis allowDecimals={false} />
        <Tooltip />
        <Legend />
        <Line type="monotone" dataKey="sign_ons" name="Sign-ons" stroke="#0B2545" strokeWidth={2} isAnimationActive={false} />
        <Line type="monotone" dataKey="sign_offs" name="Sign-offs" stroke="#13A89E" strokeWidth={2} isAnimationActive={false} />
      </LineChart>
    </ChartFrame>
  )
}

export function AllCharts({ charts }: { charts: Charts }) {
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <StatusDonut data={charts.by_status} />
      <MovementLines data={charts.movements} />
      <RankBars data={charts.by_rank} />
      <PrincipalBars data={charts.on_board_by_principal} />
    </div>
  )
}
