import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { ApiError } from '@/api/client'
import type { Vacancy } from '@/api/vacancies'
import { ToastProvider } from '@/components/ui/Toast'
import VacancyFormPage from './VacancyFormPage'

const create = vi.fn()
const update = vi.fn()

vi.mock('@/api/vacancies', () => ({
  vacanciesApi: {
    get: () => Promise.resolve(existing),
    create: (...args: unknown[]) => create(...args) as unknown,
    update: (...args: unknown[]) => update(...args) as unknown,
    setStatus: vi.fn(),
    remove: vi.fn(),
  },
}))
vi.mock('@/api/options', () => ({
  OPTIONS_KEY: ['options'],
  optionsApi: {
    all: () =>
      Promise.resolve({
        companies: [],
        vessels: [],
        ranks: [{ id: '11111111-1111-1111-1111-111111111111', name: 'Master', short_code: 'MST', department: 'deck', is_officer: true, is_active: true, sort_order: 1 }],
        vessel_types: [{ id: '22222222-2222-2222-2222-222222222222', name: 'Bulk Carrier', is_active: true, sort_order: 1 }],
        currencies: [],
        crew_statuses: [],
        allowance_types: [],
        document_types: [],
        certificate_types: [],
      }),
  },
}))

let existing: Vacancy

function vacancy(): Vacancy {
  return {
    id: 'VAC000001',
    title: 'Chief Officer',
    slug: 'chief-officer',
    rank: { id: '11111111-1111-1111-1111-111111111111', name: 'Master' },
    vessel_type: null,
    department: 'deck',
    location: 'Jakarta',
    employment_type: 'contract',
    duration: '9 months',
    salary_text: '',
    description_html: '',
    requirements_html: '',
    required_documents: ['Passport'],
    closing_date: '2026-12-31',
    status: 'draft',
    is_expired: false,
    published_at: null,
    applications_count: 3,
    new_applications_count: 1,
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-01T00:00:00Z',
  }
}

function renderPage(path: string) {
  existing = vacancy()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createMemoryRouter(
    [
      { path: '/vacancies/new', element: <VacancyFormPage /> },
      { path: '/vacancies/:id', element: <VacancyFormPage /> },
    ],
    { initialEntries: [path] },
  )
  render(
    <QueryClientProvider client={qc}>
      <ToastProvider>
        <RouterProvider router={router} />
      </ToastProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  create.mockReset()
  update.mockReset()
})

describe('VacancyFormPage', () => {
  it('shows validation errors and does not submit an empty form', async () => {
    const user = userEvent.setup()
    renderPage('/vacancies/new')
    await user.click(await screen.findByRole('button', { name: 'Create vacancy' }))
    expect(await screen.findByText('The title is required.')).toBeInTheDocument()
    expect(screen.getByText('Select a rank.')).toBeInTheDocument()
    expect(create).not.toHaveBeenCalled()
  })

  it('previews the public address derived from the title while the slug is blank', async () => {
    const user = userEvent.setup()
    renderPage('/vacancies/new')
    await user.type(await screen.findByLabelText(/^Title/), 'Third Officer & Crew')
    expect(screen.getByTestId('slug-preview')).toHaveTextContent('/career/third-officer-and-crew')
  })

  it('normalizes a typed slug in the preview', async () => {
    const user = userEvent.setup()
    renderPage('/vacancies/new')
    await user.type(await screen.findByLabelText(/^Title/), 'Ignored')
    await user.type(screen.getByLabelText('Slug'), 'My Custom  Slug!')
    expect(screen.getByTestId('slug-preview')).toHaveTextContent('/career/my-custom-slug')
  })

  it('fills the slug from the title with the Auto from title button', async () => {
    const user = userEvent.setup()
    renderPage('/vacancies/new')
    await user.type(await screen.findByLabelText(/^Title/), 'Able Seaman')
    await user.click(screen.getByRole('button', { name: 'Auto from title' }))
    expect(screen.getByLabelText('Slug')).toHaveValue('able-seaman')
  })

  it('maps server field errors onto the matching fields', async () => {
    const user = userEvent.setup()
    create.mockRejectedValue(new ApiError(422, { code: 'validation_failed', message: 'Validation failed.', fields: { closing_date: 'The closing date must be today or later for an open vacancy.' } }))
    renderPage('/vacancies/new')
    await user.type(await screen.findByLabelText(/^Title/), 'Master')
    await screen.findByRole('option', { name: 'Master' })
    await user.selectOptions(screen.getByLabelText(/^Rank/), 'Master')
    await user.click(screen.getByRole('button', { name: 'Create vacancy' }))
    expect(await screen.findByText('The closing date must be today or later for an open vacancy.')).toBeInTheDocument()
    expect(create).toHaveBeenCalledWith(expect.objectContaining({ title: 'Master', rank_id: '11111111-1111-1111-1111-111111111111', status: 'draft' }))
  })

  it('shows a general error when the server rejects a field the form does not know', async () => {
    const user = userEvent.setup()
    create.mockRejectedValue(new ApiError(422, { code: 'validation_failed', message: 'The slug is already used.', fields: { unknown_field: 'x' } }))
    renderPage('/vacancies/new')
    await user.type(await screen.findByLabelText(/^Title/), 'Master')
    await screen.findByRole('option', { name: 'Master' })
    await user.selectOptions(screen.getByLabelText(/^Rank/), 'Master')
    await user.click(screen.getByRole('button', { name: 'Create vacancy' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('The slug is already used.')
  })

  it('keeps the current slug in the preview when editing and the slug is cleared', async () => {
    const user = userEvent.setup()
    renderPage('/vacancies/VAC000001')
    const slug = await screen.findByLabelText('Slug')
    await user.clear(slug)
    expect(screen.getByTestId('slug-preview')).toHaveTextContent('/career/chief-officer')
  })

  it('sends the changed values when saving an existing vacancy', async () => {
    const user = userEvent.setup()
    update.mockResolvedValue({ ...vacancy(), title: 'Chief Officer II', updated_at: '2026-10-02T00:00:00Z' })
    renderPage('/vacancies/VAC000001')
    const title = await screen.findByLabelText(/^Title/)
    await user.clear(title)
    await user.type(title, 'Chief Officer II')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(update).toHaveBeenCalled())
    expect(update).toHaveBeenCalledWith('VAC000001', expect.objectContaining({ title: 'Chief Officer II', required_documents: ['Passport'], closing_date: '2026-12-31' }))
  })
})
