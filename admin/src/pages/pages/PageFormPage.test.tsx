import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { ApiError } from '@/api/client'
import type { PageDetail } from '@/api/pages'
import { ToastProvider } from '@/components/ui/Toast'
import PageFormPage from './PageFormPage'

const create = vi.fn()
const update = vi.fn()

vi.mock('@/api/pages', () => ({
  pagesApi: {
    get: () => Promise.resolve(existing),
    create: (...args: unknown[]) => create(...args) as unknown,
    update: (...args: unknown[]) => update(...args) as unknown,
    remove: vi.fn(),
  },
}))
vi.mock('@/api/galleryOptions', () => ({ galleryOptionsApi: { active: () => Promise.resolve([]) } }))

let existing: PageDetail

function detail(overrides: Partial<PageDetail> = {}): PageDetail {
  return {
    id: 'p1',
    title: 'About us',
    slug: 'about-us',
    path: '/about-us',
    system_key: null,
    is_system: false,
    status: 'draft',
    in_header_menu: false,
    in_footer_menu: false,
    menu_order: 0,
    published_at: null,
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-01T00:00:00Z',
    intro_html: '',
    seo_title: '',
    meta_description: '',
    og_image_file_id: null,
    og_image: null,
    media: {},
    blocks: [
      { id: 'b1', type: 'rich_text', hidden: false, settings: { html: '<p>First</p>' } },
      { id: 'b2', type: 'rich_text', hidden: false, settings: { html: '<p>Second</p>' } },
      { id: 'b3', type: 'stats', hidden: false, settings: { items: [{ number: 5, suffix: '+', label: 'Ships' }] } },
    ],
    ...overrides,
  }
}

function renderPage(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createMemoryRouter(
    [
      { path: '/website/pages/new', element: <PageFormPage /> },
      { path: '/website/pages/:id', element: <PageFormPage /> },
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

function lastPayload() {
  const call = update.mock.calls[0] as [string, { blocks: { id: string; hidden: boolean; type: string; settings: unknown }[] }]
  return call[1]
}

beforeEach(() => {
  create.mockReset()
  update.mockReset()
  existing = detail()
})

describe('PageFormPage block builder', () => {
  it('sends duplicated, reordered and hidden blocks in the saved payload', async () => {
    const user = userEvent.setup()
    update.mockResolvedValue(existing)
    renderPage('/website/pages/p1')
    await user.click(await screen.findByRole('button', { name: 'Duplicate block 1' }))
    screen.getByRole('button', { name: /^Reorder block 3\./ }).focus()
    await user.keyboard('{ArrowUp}')
    await user.click(screen.getByRole('button', { name: 'Hide block 1' }))
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(update).toHaveBeenCalled())

    const blocks = lastPayload().blocks
    expect(blocks).toHaveLength(4)
    expect(blocks.map((b) => b.type)).toEqual(['rich_text', 'rich_text', 'rich_text', 'stats'])
    expect(blocks[0]).toMatchObject({ id: 'b1', hidden: true })
    expect(blocks[1]?.id).toBe('b2')
    expect(blocks[2]?.id).not.toBe('b1')
    expect(blocks[2]?.settings).toEqual({ html: '<p>First</p>' })
    expect(blocks[3]?.id).toBe('b3')
  })

  it('adds a block, fills it in and saves it with a client-generated id', async () => {
    const user = userEvent.setup()
    update.mockResolvedValue(existing)
    renderPage('/website/pages/p1')
    await user.click(await screen.findByRole('button', { name: 'Add block' }))
    await user.click(within(screen.getByRole('list', { name: 'Block types' })).getByRole('button', { name: /Stats Counter/ }))
    const card = screen.getByRole('group', { name: 'Block 4: Stats Counter' })
    await user.type(within(card).getByLabelText(/^Label/), 'Years at sea')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(update).toHaveBeenCalled())

    const added = lastPayload().blocks[3]
    expect(added?.type).toBe('stats')
    expect(added?.id).toMatch(/^[A-Za-z0-9_-]{1,64}$/)
    expect(added?.settings).toEqual({ items: [{ number: 0, suffix: '', label: 'Years at sea' }] })
  })

  it('removes a block only after confirmation', async () => {
    const user = userEvent.setup()
    update.mockResolvedValue(existing)
    renderPage('/website/pages/p1')
    await user.click(await screen.findByRole('button', { name: 'Delete block 2' }))
    expect(screen.getByRole('button', { name: 'Delete block' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Delete block' }))
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(update).toHaveBeenCalled())
    expect(lastPayload().blocks.map((b) => b.id)).toEqual(['b1', 'b3'])
  })

  it('dims hidden blocks and marks them as hidden', async () => {
    const user = userEvent.setup()
    renderPage('/website/pages/p1')
    await user.click(await screen.findByRole('button', { name: 'Hide block 2' }))
    expect(screen.getByRole('group', { name: 'Block 2: Rich Text' })).toHaveAttribute('data-hidden', 'true')
    expect(screen.getByRole('button', { name: 'Show block 2' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('blocks saving when a block fails validation and names the problem', async () => {
    const user = userEvent.setup()
    existing = detail({ blocks: [{ id: 'g1', type: 'gallery', hidden: false, settings: { image_ids: [], columns: 3, lightbox: true } }] })
    renderPage('/website/pages/p1')
    await user.click(await screen.findByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText('Block 1 (Gallery) › Image ids: Choose at least one image.')).toBeInTheDocument()
    expect(update).not.toHaveBeenCalled()
  })

  it('shows a nested server error on the matching input and expands its block', async () => {
    const user = userEvent.setup()
    update.mockRejectedValue(
      new ApiError(422, { code: 'validation_failed', message: 'Validation failed.', fields: { 'blocks.2.settings.items.0.label': 'The label must be at most 80 characters.' } }),
    )
    renderPage('/website/pages/p1')
    await user.click(await screen.findByRole('button', { name: 'Save changes' }))
    const card = await screen.findByRole('group', { name: 'Block 3: Stats Counter' })
    const label = await within(card).findByLabelText(/^Label/)
    expect(label).toHaveAttribute('aria-invalid', 'true')
    expect(within(card).getByText('The label must be at most 80 characters.')).toBeInTheDocument()
    expect(screen.getByText('Block 3 (Stats Counter) › Items 1 › Label: The label must be at most 80 characters.')).toBeInTheDocument()
  })
})

describe('PageFormPage settings', () => {
  it('hides the builder and locks slug and status for system pages', async () => {
    update.mockResolvedValue(existing)
    existing = detail({ is_system: true, system_key: 'career', slug: 'career', path: '/career', status: 'published' })
    renderPage('/website/pages/p1')
    expect(await screen.findByText('This is a system page.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add block' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('Slug')).toBeDisabled()
    expect(screen.getByLabelText('Status')).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Delete page' })).not.toBeInTheDocument()
  })

  it('counts SEO characters and previews the search result', async () => {
    const user = userEvent.setup()
    renderPage('/website/pages/p1')
    await user.type(await screen.findByLabelText('SEO title'), 'Crew agency')
    expect(screen.getByLabelText('SEO title length')).toHaveTextContent('11/120')
    expect(screen.getByLabelText('Meta description length')).toHaveTextContent('0/320')
    expect(within(screen.getByRole('group', { name: 'Search result preview' })).getByText('Crew agency')).toBeInTheDocument()
  })

  it('previews the path from the title when creating a page', async () => {
    const user = userEvent.setup()
    renderPage('/website/pages/new')
    await user.type(await screen.findByLabelText(/^Title/), 'Our Fleet')
    expect(screen.getByTestId('slug-preview')).toHaveTextContent('/our-fleet')
  })
})
