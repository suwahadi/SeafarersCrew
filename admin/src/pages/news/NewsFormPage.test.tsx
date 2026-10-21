import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { ApiError } from '@/api/client'
import type { NewsDetail } from '@/api/news'
import { ToastProvider } from '@/components/ui/Toast'
import NewsFormPage from './NewsFormPage'
import { NewsStatusBadge } from './NewsStatusBadge'

const create = vi.fn()
const update = vi.fn()

vi.mock('@/api/news', () => ({
  newsApi: {
    get: () => Promise.resolve(existing),
    create: (...args: unknown[]) => create(...args) as unknown,
    update: (...args: unknown[]) => update(...args) as unknown,
    remove: vi.fn(),
  },
}))

let existing: NewsDetail

function article(overrides: Partial<NewsDetail> = {}): NewsDetail {
  return {
    id: 'n1',
    title: 'New vessel',
    slug: 'new-vessel',
    excerpt: 'Short',
    thumbnail_file_id: null,
    thumbnail: null,
    published_at: null,
    status: 'draft',
    is_live: false,
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-01T00:00:00Z',
    body_html: '',
    seo_title: '',
    meta_description: '',
    ...overrides,
  }
}

function renderPage(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createMemoryRouter(
    [
      { path: '/website/news/new', element: <NewsFormPage /> },
      { path: '/website/news/:id', element: <NewsFormPage /> },
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
  existing = article()
})

describe('NewsStatusBadge', () => {
  it.each([
    [{ status: 'draft', is_live: false }, 'Draft'],
    [{ status: 'published', is_live: false }, 'Scheduled'],
    [{ status: 'published', is_live: true }, 'Live'],
  ] as const)('labels %j as %s', (news, label) => {
    render(<NewsStatusBadge news={news} />)
    expect(screen.getByText(label)).toBeInTheDocument()
  })
})

describe('NewsFormPage', () => {
  it('previews the news address from the title', async () => {
    const user = userEvent.setup()
    renderPage('/website/news/new')
    await user.type(await screen.findByLabelText(/^Title/), 'Port Update')
    expect(screen.getByTestId('slug-preview')).toHaveTextContent('/news/port-update')
  })

  it('counts excerpt characters', async () => {
    const user = userEvent.setup()
    renderPage('/website/news/n1')
    await user.type(await screen.findByLabelText('Excerpt'), 'abc')
    expect(screen.getByLabelText('Excerpt length')).toHaveTextContent('8/300')
  })

  it('shows the scheduled hint and sends the publish time as UTC for a future published article', async () => {
    const user = userEvent.setup()
    update.mockResolvedValue(existing)
    renderPage('/website/news/n1')
    await user.selectOptions(await screen.findByLabelText(/^Status/), 'published')
    await user.type(screen.getByLabelText('Publish date and time'), '2999-11-01T10:00')
    expect(screen.getByText(/^Scheduled: the article stays hidden until/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(update).toHaveBeenCalled())
    expect(update).toHaveBeenCalledWith('n1', expect.objectContaining({ status: 'published', published_at: '2999-11-01T03:00:00Z', thumbnail_file_id: null }))
  })

  it('suggests publishing when a draft has a future date', async () => {
    const user = userEvent.setup()
    renderPage('/website/news/n1')
    await user.type(await screen.findByLabelText('Publish date and time'), '2999-11-01T10:00')
    expect(screen.getByText(/Set the status to Published/)).toBeInTheDocument()
  })

  it('maps a server error on the publish date to its input', async () => {
    const user = userEvent.setup()
    update.mockRejectedValue(new ApiError(422, { code: 'validation_failed', message: 'Validation failed.', fields: { published_at: 'The publish date is invalid.' } }))
    renderPage('/website/news/n1')
    await user.click(await screen.findByRole('button', { name: 'Save changes' }))
    expect(await screen.findByLabelText('Publish date and time')).toHaveAttribute('aria-invalid', 'true')
  })
})
