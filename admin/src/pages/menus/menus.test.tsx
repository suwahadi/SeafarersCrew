import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { ToastProvider } from '@/components/ui/Toast'
import { makeNode, type MenuNode } from '@/lib/menuTree'
import { mockApi } from '@/test/utils'
import MenusPage from './MenusPage'
import { MenuTreeEditor } from './MenuTreeEditor'

afterEach(() => {
  vi.unstubAllGlobals()
})

function Harness({ initial, errors = {} }: { initial: MenuNode[]; errors?: Record<string, Record<string, string>> }) {
  const [tree, setTree] = useState(initial)
  return <MenuTreeEditor tree={tree} errors={errors} onChange={setTree} />
}

const link = (label: string) => makeNode({ label, url: `/${label.toLowerCase()}` })

describe('menu tree editor', () => {
  it('indents an item under the previous one and outdents it back', async () => {
    render(<Harness initial={[link('About'), link('Team')]} />)
    await userEvent.click(screen.getByRole('button', { name: 'Indent Team' }))
    const sub = screen.getByRole('list', { name: 'Sub-items of About' })
    expect(within(sub).getByDisplayValue('Team')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Indent Team' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Outdent Team' }))
    expect(screen.queryByRole('list', { name: 'Sub-items of About' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Indent Team' })).toBeEnabled()
  })

  it('disables indent for the first item and for an item that already has sub-items', () => {
    const parent = link('Company')
    parent.children = [link('Team')]
    render(<Harness initial={[link('Home'), parent]} />)
    expect(screen.getByRole('button', { name: 'Indent Home' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Indent Company' })).toBeDisabled()
  })

  it('moves an item with the arrow keys on its drag handle and has no arrow buttons', async () => {
    render(<Harness initial={[link('A'), link('B')]} />)
    screen.getByRole('button', { name: /Reorder B/ }).focus()
    await userEvent.keyboard('{ArrowUp}')
    const inputs = screen.getAllByLabelText('Label')
    expect(inputs.map((i) => (i as HTMLInputElement).value)).toEqual(['B', 'A'])
    expect(screen.queryByRole('button', { name: /Move .* (up|down)/ })).not.toBeInTheDocument()
  })

  it('shows a draft warning for draft pages and server errors on the right row', () => {
    const page = makeNode({ label: 'Soon', pageId: 'p1', page: { title: 'Coming soon', path: '/soon', status: 'draft' } })
    const bad = link('Bad')
    render(<Harness initial={[page, bad]} errors={{ [bad.key]: { url: 'The link is not allowed.' } }} />)
    expect(screen.getByText('Draft')).toBeInTheDocument()
    expect(screen.getByText(/stays hidden on the site/)).toBeInTheDocument()
    expect(screen.getByText('The link is not allowed.')).toBeInTheDocument()
  })
})

describe('menus page', () => {
  const serverMenus = {
    header: [
      { id: 'm1', label: 'About', page_id: 'p1', url: '', open_in_new_tab: false, page: { title: 'About us', path: '/about', status: 'published' }, children: [] },
      { id: 'm2', label: 'Docs', page_id: null, url: '/docs', open_in_new_tab: false, page: null, children: [] },
    ],
    footer: [],
  }

  function renderPage() {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
    const router = createMemoryRouter([{ path: '/', element: <MenusPage /> }])
    render(
      <QueryClientProvider client={client}>
        <ToastProvider>
          <RouterProvider router={router} />
        </ToastProvider>
      </QueryClientProvider>,
    )
  }

  it('replaces the whole menu with the edited tree', async () => {
    const calls = mockApi((c) => {
      if (c.path === '/menus') return { body: { data: serverMenus } }
      if (c.path.startsWith('/pages')) return { body: { data: [], meta: { page: 1, per_page: 100, total: 0 } } }
      if (c.method === 'PUT') return { body: { data: { header: [], footer: [] } } }
      return undefined
    })
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Indent Docs' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save header menu' }))
    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    const put = calls.find((c) => c.method === 'PUT')
    expect(put?.path).toBe('/menus/header')
    expect(put?.body).toEqual({
      items: [{ label: 'About', page_id: 'p1', open_in_new_tab: false, children: [{ label: 'Docs', url: '/docs', open_in_new_tab: false }] }],
    })
  })

  it('shows a server error beside the offending child item', async () => {
    mockApi((c) => {
      if (c.path === '/menus') return { body: { data: serverMenus } }
      if (c.path.startsWith('/pages')) return { body: { data: [], meta: { page: 1, per_page: 100, total: 0 } } }
      if (c.method === 'PUT') return { status: 422, body: { error: { code: 'validation_failed', message: 'Invalid menu', fields: { 'items.0.children.0.url': 'That link is not allowed.' } } } }
      return undefined
    })
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Indent Docs' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save header menu' }))
    const sub = await screen.findByRole('list', { name: 'Sub-items of About' })
    expect(await within(sub).findByText('That link is not allowed.')).toBeInTheDocument()
  })

  it('adds a custom link after validating it', async () => {
    mockApi((c) => {
      if (c.path === '/menus') return { body: { data: { header: [], footer: [] } } }
      if (c.path.startsWith('/pages')) return { body: { data: [], meta: { page: 1, per_page: 100, total: 0 } } }
      return undefined
    })
    renderPage()
    await screen.findByText(/This menu is empty/)
    const card = screen.getByText('Add a custom link').closest('section')
    if (!card) throw new Error('missing card')
    await userEvent.type(within(card).getByLabelText('Label'), 'Blog')
    await userEvent.type(within(card).getByLabelText('URL'), 'blog')
    await userEvent.click(within(card).getByRole('button', { name: 'Add link' }))
    expect(within(card).getByText(/Use a path starting with/)).toBeInTheDocument()
    await userEvent.clear(within(card).getByLabelText('URL'))
    await userEvent.type(within(card).getByLabelText('URL'), '/blog')
    await userEvent.click(within(card).getByRole('button', { name: 'Add link' }))
    expect(screen.getByDisplayValue('Blog')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save header menu' })).toBeEnabled()
  })
})
