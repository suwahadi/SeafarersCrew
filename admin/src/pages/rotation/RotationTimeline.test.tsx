import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { Timeline } from '@/api/contracts'
import { RotationTimeline } from './RotationTimeline'

const timeline: Timeline = {
  from: '2026-10-01',
  to: '2026-12-31',
  today: '2026-10-21',
  lanes: [
    {
      vessel: { id: 'v1', name: 'MV Alpha' },
      vessel_type: 'Tug',
      bars: [
        {
          contract_id: 'AAAAAAAA1',
          crew: { id: 'C1', name: 'Budi Santoso' },
          rank_name: 'Master',
          sign_on_date: '2026-06-01',
          planned_sign_off_date: '2026-10-10',
          state: 'overdue',
          relief_plans: [],
        },
        {
          contract_id: 'AAAAAAAA2',
          crew: { id: 'C2', name: 'Agus Salim' },
          rank_name: 'Chief Engineer',
          sign_on_date: '2026-09-01',
          planned_sign_off_date: '2026-11-05',
          state: 'due_soon',
          relief_plans: [{ id: 'p1', reliever: { id: 'C3', name: 'Rudi Hartono' }, planned_relief_date: '2026-11-05', joining_port: 'Batam', status: 'planned' }],
        },
        {
          contract_id: 'AAAAAAAA3',
          crew: { id: 'C4', name: 'Dewi Lestari' },
          rank_name: 'Cook',
          sign_on_date: '2026-10-15',
          planned_sign_off_date: '2027-04-15',
          state: 'normal',
          relief_plans: [],
        },
        {
          contract_id: 'AAAAAAAA4',
          crew: { id: 'C5', name: 'Outside Range' },
          rank_name: 'Deckhand',
          sign_on_date: '2025-01-01',
          planned_sign_off_date: '2025-06-01',
          state: 'normal',
          relief_plans: [],
        },
      ],
    },
  ],
}

describe('RotationTimeline', () => {
  it('marks each bar with its state', () => {
    render(<RotationTimeline timeline={timeline} onSelectBar={vi.fn()} onSelectPlan={vi.fn()} />)
    expect(screen.getByRole('button', { name: /Budi Santoso/ })).toHaveAttribute('data-state', 'overdue')
    expect(screen.getByRole('button', { name: /Agus Salim/ })).toHaveAttribute('data-state', 'due_soon')
    expect(screen.getByRole('button', { name: /Dewi Lestari/ })).toHaveAttribute('data-state', 'normal')
  })

  it('colours overdue bars red and due-soon bars orange', () => {
    render(<RotationTimeline timeline={timeline} onSelectBar={vi.fn()} onSelectPlan={vi.fn()} />)
    expect(screen.getByRole('button', { name: /Budi Santoso/ }).className).toContain('bg-red-600')
    expect(screen.getByRole('button', { name: /Agus Salim/ }).className).toContain('bg-orange-600')
  })

  it('positions bars by date and clips them to the range', () => {
    render(<RotationTimeline timeline={timeline} onSelectBar={vi.fn()} onSelectPlan={vi.fn()} />)
    const overdue = screen.getByRole('button', { name: /Budi Santoso/ })
    expect(overdue.style.left).toBe('0%')
    expect(parseFloat(overdue.style.width)).toBeCloseTo((10 / 92) * 100, 1)
    const later = screen.getByRole('button', { name: /Dewi Lestari/ })
    expect(parseFloat(later.style.left)).toBeCloseTo((14 / 92) * 100, 1)
    expect(later.style.width).not.toBe('')
  })

  it('does not render bars outside the range', () => {
    render(<RotationTimeline timeline={timeline} onSelectBar={vi.fn()} onSelectPlan={vi.fn()} />)
    expect(screen.queryByRole('button', { name: /Outside Range/ })).not.toBeInTheDocument()
  })

  it('reports clicks on bars and relief markers', async () => {
    const user = userEvent.setup()
    const onSelectBar = vi.fn()
    const onSelectPlan = vi.fn()
    render(<RotationTimeline timeline={timeline} onSelectBar={onSelectBar} onSelectPlan={onSelectPlan} />)
    await user.click(screen.getByRole('button', { name: /Agus Salim/ }))
    expect(onSelectBar).toHaveBeenCalledWith(expect.objectContaining({ contract_id: 'AAAAAAAA2' }), 'MV Alpha')
    await user.click(screen.getByRole('button', { name: /Relief by Rudi Hartono/ }))
    expect(onSelectPlan).toHaveBeenCalledWith(expect.objectContaining({ id: 'p1' }), expect.objectContaining({ contract_id: 'AAAAAAAA2' }), 'MV Alpha')
  })
})
