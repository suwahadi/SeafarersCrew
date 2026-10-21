import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { DataGrid, type GridPage, type GridParams } from './DataGrid'

interface Row {
  id: string
  name: string
  rank: string
}

const columns: ColumnDef<Row>[] = [
  { id: 'full_name', header: 'Name', accessorKey: 'name' },
  { id: 'rank', header: 'Rank', accessorKey: 'rank', enableSorting: false },
]

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

function setup(rows: Row[] = [{ id: '1', name: 'Budi', rank: 'Master' }]) {
  const calls: GridParams[] = []
  const fetchPage = vi.fn((params: GridParams): Promise<GridPage<Row>> => {
    calls.push(params)
    return Promise.resolve({ rows, total: 45 })
  })
  render(<DataGrid id="test" columns={columns} fetchPage={fetchPage} rowKey={(r) => r.id} />, { wrapper })
  return { calls, fetchPage }
}

describe('DataGrid', () => {
  it('requests the first page with the default page size and renders rows', async () => {
    const { calls } = setup()
    expect(await screen.findByText('Budi')).toBeInTheDocument()
    expect(calls[0]).toMatchObject({ page: 1, per_page: 20 })
    expect(screen.getByText('1–20 of 45')).toBeInTheDocument()
  })

  it('debounces the search and resets to the first page', async () => {
    const user = userEvent.setup()
    const { calls } = setup()
    await screen.findByText('Budi')
    await user.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() => expect(calls.at(-1)?.page).toBe(2))
    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'bud')
    await waitFor(() => expect(calls.at(-1)).toMatchObject({ q: 'bud', page: 1 }))
    expect(calls.filter((c) => c.q === 'b' || c.q === 'bu')).toHaveLength(0)
  })

  it('sends server-side sort parameters', async () => {
    const user = userEvent.setup()
    const { calls } = setup()
    await screen.findByText('Budi')
    await user.click(screen.getByRole('button', { name: /Name/ }))
    await waitFor(() => expect(calls.at(-1)).toMatchObject({ sort: 'full_name', order: 'asc' }))
  })

  it('remembers hidden columns per browser', async () => {
    const user = userEvent.setup()
    setup()
    await screen.findByText('Budi')
    await user.click(screen.getByRole('button', { name: 'Columns' }))
    await user.click(screen.getByRole('checkbox', { name: 'Rank' }))
    expect(screen.queryByText('Master')).not.toBeInTheDocument()
    expect(JSON.parse(window.localStorage.getItem('grid:test:columns') ?? '{}')).toEqual({ rank: false })
  })

  it('shows an empty message', async () => {
    setup([])
    expect(await screen.findByText('No records found.')).toBeInTheDocument()
  })
})
