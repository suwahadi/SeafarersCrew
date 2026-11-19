import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import type { CvPreview } from '@/api/crewCv'
import { ToastProvider } from '@/components/ui/Toast'
import { mockApi } from '@/test/utils'
import CrewImportPage from './CrewImportPage'

const options = {
  companies: [],
  vessels: [],
  currencies: [],
  crew_statuses: [],
  allowance_types: [],
  ranks: [{ id: 'rank-ab', name: 'Able Seaman', short_code: 'AB', department: 'deck', is_officer: false, is_active: true, sort_order: 1 }],
  vessel_types: [{ id: 'vt-ahts', name: 'AHTS', is_active: true, sort_order: 1 }],
  document_types: [{ id: 'doc-pass', code: 'passport', name: 'Passport', is_active: true, sort_order: 1 }],
  certificate_types: [{ id: 'cert-bst', code: 'bst', name: 'Basic Safety Training', is_active: true, sort_order: 1 }],
}

function preview(extra: Partial<CvPreview> = {}): CvPreview {
  return {
    token: 'tok',
    full_name: 'BUDI SANTOSO',
    personal: [
      { label: 'Name', value: 'BUDI SANTOSO' },
      { label: 'Rank', value: 'XX', choice_key: 'rank:XX' },
    ],
    next_of_kin: [{ row: 0, cells: { name: 'SITI AMINAH', relationship: 'spouse', relationship_text: 'WIFE', phone: '+6281211112222', address: '' } }],
    documents: [{ row: 23, cells: { type: 'Passport', number: 'X 1234567', issued: '2020-01-10', expires: '2025-01-10', place: 'JAKARTA' }, matched_as: 'Passport' }],
    certificates: [{ row: 36, cells: { text: 'Basic Safety Training / BST', issued: '2019-02-01', expires: '2024-02-01' }, matched_as: 'Basic Safety Training' }],
    sea_service: [
      {
        row: 5,
        cells: { vessel: 'MV. NUSA 1', rank: 'XX', flag: 'INDONESIA', vessel_type: 'AHTS', grt_hp: '1500', company: 'PT. CONTOH', from: '2019-01-01', to: '2019-06-30' },
        choice_keys: ['rank:XX'],
      },
    ],
    not_imported: ['Blood group'],
    warnings: [],
    errors: [],
    choices: [{ key: 'rank:XX', kind: 'rank', text: 'XX', allow_create: false }],
    ...extra,
  }
}

const cvFile = () => new File(['xlsx'], 'CV RVOS 2024 - AB - BUDI.xlsx', { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })

function renderPanel(reply: { status?: number; body: unknown }) {
  const calls = mockApi((call) => {
    if (call.path === '/options') return { body: { data: options } }
    if (call.path === '/crews/cv-import/validate') return reply
    if (call.path === '/crews/cv-import/commit') return { body: { data: { crew_id: 'ABCDEFGH2', full_name: 'BUDI SANTOSO' } } }
    return undefined
  })
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  const router = createMemoryRouter(
    [
      { path: '/crews/import', element: <CrewImportPage /> },
      { path: '/crews/:id', element: <p>Profile page</p> },
    ],
    { initialEntries: ['/crews/import?tab=cv'] },
  )
  render(
    <QueryClientProvider client={qc}>
      <ToastProvider>
        <RouterProvider router={router} />
      </ToastProvider>
    </QueryClientProvider>,
  )
  return { calls, user: userEvent.setup() }
}

describe('CvImportPanel', () => {
  it('previews the CV and confirms only after every choice is made', async () => {
    const { calls, user } = renderPanel({ body: { data: preview() } })
    await user.upload(screen.getByLabelText('CV file (.xlsx)'), cvFile())

    expect(await screen.findByText('1 crew · 1 document · 1 certificate · 1 sea service (1 needs a choice) · 1 next of kin')).toBeInTheDocument()
    expect(screen.getAllByText('BUDI SANTOSO').length).toBeGreaterThan(0)
    expect(screen.getByText('Blood group')).toBeInTheDocument()
    expect(screen.getAllByText('Needs a choice').length).toBeGreaterThan(0)
    const confirm = screen.getByRole('button', { name: 'Confirm import' })
    expect(confirm).toBeDisabled()

    await user.selectOptions(screen.getByLabelText('Rank in CV: XX'), 'match:rank-ab')
    expect(confirm).toBeEnabled()
    await user.click(confirm)

    expect(await screen.findByText('Profile page')).toBeInTheDocument()
    expect(screen.getByText('Imported BUDI SANTOSO from CV.')).toBeInTheDocument()
    expect(calls.find((c) => c.path === '/crews/cv-import/commit')?.body).toEqual({
      token: 'tok',
      decisions: { 'rank:XX': { action: 'match', target_id: 'rank-ab' } },
    })
  })

  it('stops with a link to the existing profile when the crew is registered', async () => {
    const { user } = renderPanel({ body: { data: preview({ token: '', duplicate: { crew_id: 'OLDCREW02', full_name: 'Budi Santoso', reason: 'passport' } }) } })
    await user.upload(screen.getByLabelText('CV file (.xlsx)'), cvFile())

    expect(await screen.findByText('Budi Santoso is already registered (same passport number).')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open profile' })).toHaveAttribute('href', '/crews/OLDCREW02')
    expect(screen.queryByRole('button', { name: 'Confirm import' })).not.toBeInTheDocument()
  })

  it('lists unreadable cells and keeps confirm disabled', async () => {
    const { user } = renderPanel({ body: { data: preview({ token: '', choices: [], errors: [{ section: 'certificates', row: 36, message: 'The date could not be read.' }] }) } })
    await user.upload(screen.getByLabelText('CV file (.xlsx)'), cvFile())

    expect(await screen.findByText('Certificates row 36: The date could not be read.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirm import' })).toBeDisabled()
  })

  it('shows why a file was rejected', async () => {
    const { user } = renderPanel({
      status: 422,
      body: { error: { code: 'validation_failed', message: 'The given data is invalid.', fields: { file: 'Save the CV as .xlsx and upload it again.' } } },
    })
    await user.upload(screen.getByLabelText('CV file (.xlsx)'), cvFile())
    expect(await screen.findByText('Save the CV as .xlsx and upload it again.')).toBeInTheDocument()
  })
})

describe('CrewImportPage tabs', () => {
  it('opens the CV tab from the URL and switches back to CSV', async () => {
    const { user } = renderPanel({ body: { data: preview() } })
    expect(screen.getByRole('tab', { name: 'CV (one crew)' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByLabelText('CV file (.xlsx)')).toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: 'CSV (many crew)' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Download template' })).toBeInTheDocument())
    expect(screen.queryByLabelText('CV file (.xlsx)')).not.toBeInTheDocument()
  })
})
