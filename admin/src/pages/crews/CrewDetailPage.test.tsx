import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import type { Crew } from '@/api/crews'
import { ToastProvider } from '@/components/ui/Toast'
import CrewDetailPage from './CrewDetailPage'

const crew: Crew = {
  id: 'ABC123XYZ',
  full_name: 'Budi Santoso',
  place_of_birth: 'Jakarta',
  date_of_birth: '1990-05-01',
  age: 36,
  gender: 'male',
  nationality: 'Indonesian',
  religion: '',
  marital_status: null,
  height_cm: null,
  weight_kg: null,
  shoe_size: '',
  coverall_size: '',
  applied_rank_id: null,
  applied_rank_name: 'Master',
  status: { code: 'available', name: 'Available', color: '#16a34a' },
  status_reason: '',
  photo_file_id: null,
  cv_file_id: null,
  address: '',
  city: '',
  province: '',
  country: '',
  postal_code: '',
  home_phone: '',
  mobile_phone: '+628111111111',
  email: '',
  nearest_airport: '',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const { update } = vi.hoisted(() => ({ update: vi.fn() }))

vi.mock('@/api/crews', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/api/crews')>()
  return {
    ...original,
    crewsApi: {
      ...original.crewsApi,
      get: () => Promise.resolve(crew),
      update,
      compliance: () => Promise.resolve({ rank_id: null, check_date: '2026-10-21', compliant: true, items: [] }),
      currentAssignment: () => Promise.resolve(null),
    },
  }
})

vi.mock('@/api/options', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/api/options')>()
  return {
    ...original,
    optionsApi: {
      all: () =>
        Promise.resolve({ companies: [], vessels: [], ranks: [], vessel_types: [], currencies: [], crew_statuses: [], allowance_types: [], document_types: [], certificate_types: [] }),
    },
  }
})

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createMemoryRouter([{ path: '/crews/:id', element: <CrewDetailPage /> }], { initialEntries: ['/crews/ABC123XYZ'] })
  render(
    <QueryClientProvider client={qc}>
      <ToastProvider>
        <RouterProvider router={router} />
      </ToastProvider>
    </QueryClientProvider>,
  )
}

describe('CrewDetailPage tabs', () => {
  beforeEach(() => {
    update.mockReset()
  })
  it('marks only the edited tab as dirty and saves it independently of the other', async () => {
    update.mockImplementation((_id: string, input: { full_name: string }) => Promise.resolve({ ...crew, full_name: input.full_name }))
    const user = userEvent.setup()
    update.mockImplementation((_id: string, input: { full_name: string }) => Promise.resolve({ ...crew, full_name: input.full_name }))
    renderPage()

    await screen.findByRole('heading', { name: 'Budi Santoso' })
    const personalTab = screen.getByRole('tab', { name: /Personal Data/ })
    const contactTab = screen.getByRole('tab', { name: /Contact & Address/ })
    expect(personalTab.querySelector('[aria-label="unsaved changes"]')).toBeNull()

    const fullName = screen.getByLabelText(/Full name/)
    await user.clear(fullName)
    await user.type(fullName, 'Budi Hartono')
    await waitFor(() => expect(personalTab.querySelector('[aria-label="unsaved changes"]')).not.toBeNull())
    expect(contactTab.querySelector('[aria-label="unsaved changes"]')).toBeNull()

    await user.click(contactTab)
    const mobile = screen.getByLabelText('Mobile phone')
    await user.clear(mobile)
    await user.type(mobile, '+628222222222')
    await waitFor(() => expect(contactTab.querySelector('[aria-label="unsaved changes"]')).not.toBeNull())

    await user.click(personalTab)
    await user.click(screen.getByRole('button', { name: 'Save personal data' }))

    await waitFor(() => expect(update).toHaveBeenCalledTimes(1))
    expect(update.mock.calls[0]?.[1]).toMatchObject({ full_name: 'Budi Hartono', mobile_phone: '+628111111111' })
    await waitFor(() => expect(personalTab.querySelector('[aria-label="unsaved changes"]')).toBeNull())
    expect(contactTab.querySelector('[aria-label="unsaved changes"]')).not.toBeNull()
  })
})
