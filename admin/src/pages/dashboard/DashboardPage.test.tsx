import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import type { DashboardSummary, ExpiringItem, UpcomingRelief } from '@/api/dashboard'
import { monthLabel, rankRows } from './DashboardCharts'
import { ExpiringTable, KpiCards, ReliefsTable } from './DashboardPage'

const summary: DashboardSummary = {
  today: '2026-10-21',
  total_crew: 60,
  by_status: [
    { key: 'available', label: 'Available', count: 30 },
    { key: 'on_board', label: 'On Board', count: 15 },
  ],
  active_contracts: 15,
  documents: { expired: 2, within_1: 3, within_2: 1, within_3: 0 },
  certificates: { expired: 1, within_1: 0, within_2: 2, within_3: 4 },
  due_for_relief: 5,
  new_applicants: 4,
  unread_messages: 2,
  thresholds: [30, 60, 90],
  last_backup_status: 'failed',
}

describe('Dashboard', () => {
  it('adds documents and certificates per threshold and links to filtered crew lists', () => {
    render(<KpiCards s={summary} />, { wrapper: MemoryRouter })
    expect(screen.getByRole('link', { name: /Expired items 3/ })).toHaveAttribute('href', '/crews?doc_validity=expired')
    expect(screen.getByRole('link', { name: /Expiring ≤ 30 d 3/ })).toHaveAttribute('href', '/crews?doc_validity=expiring&doc_days=30')
    expect(screen.getByRole('link', { name: /Expiring 61–90 d 4/ })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /On Board 15/ })).toHaveAttribute('href', '/crews?status=on_board')
    expect(screen.getByRole('alert')).toHaveTextContent('The last database backup failed')
  })

  it('lists expiring items with their badge and crew link', () => {
    const items: ExpiringItem[] = [
      { kind: 'certificate', crew_id: 'ABCDEFGH2', crew_name: 'Budi', rank_name: 'Master', type_name: 'BST', number: 'B-1', expired_date: '2026-11-01', days_left: 11, badge: 'red' },
    ]
    render(<ExpiringTable items={items} />, { wrapper: MemoryRouter })
    expect(screen.getByRole('link', { name: 'Budi' })).toHaveAttribute('href', '/crews/ABCDEFGH2')
    expect(screen.getByText('01-Nov-2026')).toBeInTheDocument()
    expect(screen.getByText('11 days')).toBeInTheDocument()
  })

  it('marks overdue reliefs', () => {
    const items: UpcomingRelief[] = [
      {
        contract_id: 'CONTRACT2', crew_id: 'ABCDEFGH2', crew_name: 'Budi', rank_name: 'Master', vessel_name: 'MV Ocean', principal_name: 'Alpha',
        planned_sign_off: '2026-10-10', relief_plan_id: null, planned_relief_date: null, relief_status: '', reliever_id: '', reliever_name: '',
        due_date: '2026-10-10', days_left: -11, state: 'overdue',
      },
    ]
    render(<ReliefsTable items={items} />, { wrapper: MemoryRouter })
    expect(screen.getByText('Overdue')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'MV Ocean' })).toHaveAttribute('href', '/contracts/CONTRACT2')
  })

  it('labels chart months and splits rank bars by department', () => {
    expect(monthLabel('2026-01')).toBe('Jan 26')
    expect(
      rankRows([
        { key: 'a', label: 'Master', group: 'deck', count: 3 },
        { key: 'b', label: 'Oiler', group: 'engine', count: 0 },
        { key: 'c', label: 'Cook', group: 'catering_other', count: 1 },
      ]),
    ).toEqual([{ label: 'Master', deck: 3 }, { label: 'Cook', catering_other: 1 }])
  })
})
