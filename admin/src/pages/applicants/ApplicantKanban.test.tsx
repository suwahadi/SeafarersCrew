import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import type { Applicant, Stage } from '@/api/applicants'
import { ToastProvider } from '@/components/ui/Toast'
import { ApplicantKanban } from './ApplicantKanban'

const setStage = vi.fn()

function applicant(id: string, name: string, stage: Stage): Applicant {
  return {
    id,
    vacancy: { id: 'VAC000001', title: 'Chief Officer', slug: 'chief-officer', status: 'open' },
    full_name: name,
    email: `${id}@example.com`,
    phone: '',
    nationality: 'Indonesian',
    rank_applied: { id: 'r', name: 'Chief Officer' },
    years_experience: 4,
    stage,
    files_count: 2,
    created_at: '2026-10-01T03:00:00Z',
  }
}

const all = [applicant('APP000001', 'Budi Santoso', 'new'), applicant('APP000002', 'Siti Aminah', 'new'), applicant('APP000003', 'Agus Salim', 'interview')]

vi.mock('@/api/applicants', () => ({
  applicantsApi: {
    stageCounts: () => Promise.resolve({ new: 2, screening: 0, interview: 1, accepted: 0, rejected: 0 }),
    list: (query: { stage: Stage }) => {
      const data = all.filter((a) => a.stage === query.stage)
      return Promise.resolve({ data, meta: { page: 1, per_page: 50, total: data.length } })
    },
    setStage: (...args: unknown[]) => setStage(...args) as unknown,
  },
}))

function renderKanban() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <ToastProvider>
        <MemoryRouter>
          <ApplicantKanban filters={{}} />
        </MemoryRouter>
      </ToastProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => setStage.mockReset())

describe('ApplicantKanban', () => {
  it('shows each applicant in the column of their stage with the stage counts', async () => {
    renderKanban()
    const newColumn = await screen.findByRole('region', { name: 'New' })
    await within(newColumn).findByText('Budi Santoso')
    expect(within(newColumn).getByText('Siti Aminah')).toBeInTheDocument()
    expect(within(newColumn).getByLabelText('2 applicants')).toBeInTheDocument()
    const interview = screen.getByRole('region', { name: 'Interview' })
    expect(await within(interview).findByText('Agus Salim')).toBeInTheDocument()
    expect(within(interview).queryByText('Budi Santoso')).not.toBeInTheDocument()
  })

  it('moves an applicant through the Move to menu and the note dialog', async () => {
    const user = userEvent.setup()
    setStage.mockResolvedValue({ ...all[0], stage: 'screening', files: [], stage_logs: [], message: '', date_of_birth: null, consent_at: null, ip_address: '', updated_at: '' })
    renderKanban()
    await screen.findByText('Budi Santoso')
    await user.selectOptions(screen.getByLabelText('Move Budi Santoso to another stage'), 'Screening')
    await user.type(await screen.findByLabelText(/^Note/), 'Looks good')
    await user.click(screen.getByRole('button', { name: 'Move applicant' }))
    await waitFor(() => expect(setStage).toHaveBeenCalledWith('APP000001', 'screening', 'Looks good'))
  })

  it('blocks a rejection without a note', async () => {
    const user = userEvent.setup()
    renderKanban()
    await screen.findByText('Budi Santoso')
    await user.selectOptions(screen.getByLabelText('Move Budi Santoso to another stage'), 'Rejected')
    await user.click(await screen.findByRole('button', { name: 'Move applicant' }))
    expect(screen.getByText('A note is required when rejecting an applicant.')).toBeInTheDocument()
    expect(setStage).not.toHaveBeenCalled()
  })

  it('does not offer the current stage as a move target', async () => {
    renderKanban()
    await screen.findByText('Agus Salim')
    const select = screen.getByLabelText('Move Agus Salim to another stage')
    expect(within(select).queryByRole('option', { name: 'Interview' })).not.toBeInTheDocument()
  })
})
