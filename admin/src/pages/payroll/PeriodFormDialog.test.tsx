import { fireEvent, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderApp } from '@/test/utils'
import { PeriodFormDialog } from './PeriodFormDialog'

describe('PeriodFormDialog', () => {
  it('validates the period', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderApp(<PeriodFormDialog onSubmit={onSubmit} onClose={() => undefined} />)

    await user.click(screen.getByRole('button', { name: 'Save period' }))
    expect(await screen.findByText('The title is required.')).toBeInTheDocument()

    await user.type(screen.getByLabelText(/Title/), 'Salary September 2026')
    fireEvent.change(screen.getByLabelText(/Period start/), {
      target: { value: '2026-09-24' },
    })
    fireEvent.change(screen.getByLabelText(/Period end/), {
      target: { value: '2026-08-25' },
    })
    await user.click(screen.getByRole('button', { name: 'Save period' }))
    expect(await screen.findByText('The end date must be on or after the start date.')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()

    fireEvent.change(screen.getByLabelText(/Period start/), {
      target: { value: '2026-08-25' },
    })
    fireEvent.change(screen.getByLabelText(/Period end/), {
      target: { value: '2026-09-24' },
    })
    await user.click(screen.getByRole('button', { name: 'Save period' }))
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1))
    expect(onSubmit).toHaveBeenCalledWith({
      title: 'Salary September 2026',
      period_start: '2026-08-25',
      period_end: '2026-09-24',
      transfer_date: null,
    })
  })

  it('starts from an existing period when editing', () => {
    renderApp(
      <PeriodFormDialog
        period={{
          id: 'PERAAAAA2',
          title: 'Salary September 2026',
          period_start: '2026-08-25',
          period_end: '2026-09-24',
          transfer_date: '2026-10-20',
          line_count: 0,
          total_usd: '0.00',
          total_idr: '0.00',
          created_at: '',
          updated_at: '',
        }}
        onSubmit={vi.fn()}
        onClose={() => undefined}
      />,
    )
    expect(screen.getByRole('heading', { name: 'Edit period' })).toBeInTheDocument()
    expect(screen.getByLabelText(/Transfer date/)).toHaveValue('2026-10-20')
  })
})
