import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router'
import type { Contract } from '@/api/contracts'
import type { ContractStatus } from '@/lib/contractRules'
import { ToastProvider } from '@/components/ui/Toast'
import ContractFormPage from './ContractFormPage'

let current: Contract

vi.mock('@/api/contracts', () => ({ contractsApi: { get: () => Promise.resolve(current) } }))
vi.mock('@/api/options', () => ({
  OPTIONS_KEY: ['options'],
  optionsApi: {
    all: () =>
      Promise.resolve({
        companies: [],
        vessels: [],
        ranks: [],
        vessel_types: [],
        currencies: [{ id: 'c', code: 'USD', name: 'US Dollar', symbol: '$', decimals: 2, is_active: true, sort_order: 1 }],
        crew_statuses: [],
        allowance_types: [],
        document_types: [],
        certificate_types: [],
      }),
  },
}))
vi.mock('@/api/documents', () => ({ documentsApi: { history: () => Promise.resolve([]) }, KIND_LABELS: {} }))
vi.mock('@/api/email', () => ({ emailApi: { logs: () => Promise.resolve({ data: [], meta: { page: 1, per_page: 10, total: 0 } }), templates: () => Promise.resolve([]) } }))

function contract(status: ContractStatus): Contract {
  return {
    id: 'ABCDEFGH1',
    crew: { id: 'CREW00001', name: 'Budi Santoso' },
    vessel: { id: 'v', name: 'MV Alpha' },
    rank: { id: 'r', name: 'Master' },
    principal: { id: 'p', name: 'Principal' },
    employer: { id: 'e', name: 'Employer' },
    sign_on_date: '2026-06-01',
    duration_months: 6,
    planned_sign_off_date: '2026-12-01',
    actual_sign_off_date: null,
    point_of_hire: 'Jakarta',
    sign_on_port: 'Batam',
    sign_off_port: 'Batam',
    currency_code: 'USD',
    currency_decimals: 2,
    basic_salary: '2000.00',
    working_hours_week: null,
    leave_days_month: null,
    probation_months: null,
    status,
    termination_reason: '',
    notes: '',
    allowances: [],
    total_monthly_pay: '2000.00',
    created_at: '2026-05-01T00:00:00Z',
    updated_at: `2026-05-01T00:00:00Z-${status}`,
  }
}

function renderPage(status: ContractStatus) {
  current = contract(status)
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>
      <ToastProvider>
        <MemoryRouter initialEntries={['/contracts/ABCDEFGH1']}>
          <Routes>
            <Route path="/contracts/:id" element={children} />
          </Routes>
        </MemoryRouter>
      </ToastProvider>
    </QueryClientProvider>
  )
  render(<ContractFormPage />, { wrapper })
}

describe('ContractFormPage edit locks', () => {
  it('keeps every field editable on a draft', async () => {
    renderPage('draft')
    expect(await screen.findByLabelText(/Basic salary/)).toBeEnabled()
    expect(screen.getByLabelText(/Sign-on date/)).toBeEnabled()
  })

  it('only allows planned sign-off, ports and notes on an active contract', async () => {
    renderPage('active')
    expect(await screen.findByLabelText(/Basic salary/)).toBeDisabled()
    expect(screen.getByLabelText(/Sign-on date/)).toBeDisabled()
    expect(screen.getByLabelText('Planned sign-off')).toBeEnabled()
    expect(screen.getByLabelText('Place of sign-on')).toBeEnabled()
    expect(screen.getByLabelText('Place of sign-off')).toBeEnabled()
    expect(screen.getByLabelText('Notes')).toBeEnabled()
  })

  it.each<ContractStatus>(['completed', 'terminated'])('is read-only for a %s contract', async (status) => {
    renderPage(status)
    expect(await screen.findByLabelText(/Basic salary/)).toBeDisabled()
    expect(screen.getByLabelText('Notes')).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Save contract' })).not.toBeInTheDocument()
  })
})
