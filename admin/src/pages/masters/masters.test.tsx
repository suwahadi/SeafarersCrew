import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ApiError } from '@/api/client'
import type { ComplianceMatrix, MasterRecord } from '@/api/masters'
import { mockApi, renderApp } from '@/test/utils'
import { masterConfigs } from './config'
import { MatrixEditor } from './ComplianceMatrixPage'
import { DeleteMessage } from './MasterPage'
import { MasterForm } from './MasterFormModal'
import { ReorderList } from './ReorderModal'
import { moveItem } from './reorder'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('master form validation', () => {
  it('rejects a vessel IMO number with a wrong check digit and does not call the API', async () => {
    const calls = mockApi(() => ({ body: { data: { vessel_types: [], companies: [] } } }))
    renderApp(<MasterForm config={masterConfigs.vessels} row={null} onDone={() => undefined} />)
    const user = userEvent.setup()
    await user.type(screen.getByLabelText(/^Name/), 'MV Ocean Star')
    await user.type(screen.getByLabelText(/^IMO number/), '9074728')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText(/valid check digit/)).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'POST')).toBe(false)
  })

  it('requires a name', async () => {
    const calls = mockApi(() => undefined)
    renderApp(<MasterForm config={masterConfigs['vessel-types']} row={null} onDone={() => undefined} />)
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('This field is required.')).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'POST')).toBe(false)
  })

  it('posts a valid vessel type and refreshes the shared options', async () => {
    const onDone = vi.fn()
    const calls = mockApi((c) => (c.method === 'POST' ? { status: 201, body: { data: { id: '1' } } } : { body: { data: [], meta: {} } }))
    const { client } = renderApp(<MasterForm config={masterConfigs['vessel-types']} row={null} onDone={onDone} />)
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    await userEvent.type(screen.getByLabelText(/^Name/), 'AHTS')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(onDone).toHaveBeenCalled())
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/vessel-types')
    expect(post?.body).toMatchObject({ name: 'AHTS', is_active: true })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['options'] })
  })

  it('shows server field errors next to the field', async () => {
    mockApi(() => ({ status: 422, body: { error: { code: 'validation_failed', message: 'Invalid', fields: { name: 'The name already exists.' } } } }))
    renderApp(<MasterForm config={masterConfigs['vessel-types']} row={null} onDone={() => undefined} />)
    await userEvent.type(screen.getByLabelText(/^Name/), 'AHTS')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('The name already exists.')).toBeInTheDocument()
  })
})

describe('delete message', () => {
  const row = { id: '1', name: 'MV Alpha', is_active: true, sort_order: 1, created_at: '', updated_at: '' } as MasterRecord

  it('lists the records that still use the master when the server answers in_use', () => {
    const error = new ApiError(409, { code: 'in_use', message: 'In use', fields: { contracts: '3', crew: '2' } })
    renderApp(<DeleteMessage config={masterConfigs.vessels} row={row} error={error} />)
    expect(screen.getByText(/MV Alpha is still in use/)).toBeInTheDocument()
    expect(screen.getByText('3 contracts')).toBeInTheDocument()
    expect(screen.getByText('2 crew')).toBeInTheDocument()
  })

  it('shows the server message for system records', () => {
    const error = new ApiError(409, { code: 'system_record', message: 'System records cannot be deleted.' })
    renderApp(<DeleteMessage config={masterConfigs['crew-statuses']} row={row} error={error} />)
    expect(screen.getByRole('alert')).toHaveTextContent('System records cannot be deleted.')
  })
})

describe('reorder', () => {
  it('moveItem moves an element and ignores out of range moves', () => {
    expect(moveItem(['a', 'b', 'c'], 0, 2)).toEqual(['b', 'c', 'a'])
    expect(moveItem(['a', 'b', 'c'], 2, 0)).toEqual(['c', 'a', 'b'])
    expect(moveItem(['a', 'b'], 0, 5)).toEqual(['a', 'b'])
  })

  it('patches the reorder endpoint with the ids in the new order', async () => {
    const rows = ['r1', 'r2', 'r3'].map((id, i) => ({ id, name: `Rank ${i + 1}`, is_active: true, sort_order: i }))
    const calls = mockApi((c) => {
      if (c.method === 'PATCH') return { body: { data: null } }
      return { body: { data: rows, meta: { page: 1, per_page: 100, total: 3 } } }
    })
    renderApp(<ReorderList config={masterConfigs.ranks} onDone={() => undefined} />)
    const user = userEvent.setup()
    ;(await screen.findByRole('button', { name: /^Reorder Rank 1\./ })).focus()
    await user.keyboard('{ArrowDown}')
    await waitFor(() => expect(screen.getByRole('button', { name: /^Reorder Rank 1\./ })).toHaveFocus())
    screen.getByRole('button', { name: /^Reorder Rank 3\./ }).focus()
    await user.keyboard('{ArrowUp}')
    await user.click(screen.getByRole('button', { name: 'Save order' }))
    await waitFor(() => expect(calls.some((c) => c.method === 'PATCH')).toBe(true))
    const patch = calls.find((c) => c.method === 'PATCH')
    expect(patch?.path).toBe('/ranks/reorder')
    expect(patch?.body).toEqual({ ids: ['r2', 'r3', 'r1'] })
  })
})

describe('compliance matrix', () => {
  const matrix: ComplianceMatrix = {
    ranks: [
      { id: 'rk1', name: 'Master', short_code: 'MST', department: 'deck', is_officer: true },
      { id: 'rk2', name: 'Oiler', short_code: 'OIL', department: 'engine', is_officer: false },
    ] as ComplianceMatrix['ranks'],
    document_types: [{ id: 'd1', name: 'Passport' }] as ComplianceMatrix['document_types'],
    certificate_types: [{ id: 'c1', name: 'BST', stcw_reference: 'A-VI/1' }] as ComplianceMatrix['certificate_types'],
    cells: [{ rank_id: 'rk1', document_type_id: 'd1', certificate_type_id: null }],
  }

  it('toggles cells, flags unsaved changes and saves the whole matrix', async () => {
    const calls = mockApi((c) => (c.method === 'PUT' ? { body: { data: matrix } } : undefined))
    renderApp(<MatrixEditor matrix={matrix} />)
    const user = userEvent.setup()
    expect(screen.getByRole('button', { name: 'Save matrix' })).toBeDisabled()
    expect(screen.getByLabelText('Master: Passport')).toBeChecked()

    await user.click(screen.getByLabelText('Oiler: BST (A-VI/1)'))
    expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
    await user.click(screen.getByLabelText('Require Passport for all ranks'))
    await user.click(screen.getByRole('button', { name: 'Save matrix' }))

    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    const put = calls.find((c) => c.method === 'PUT')
    expect(put?.path).toBe('/compliance-matrix')
    expect(put?.body).toEqual({
      cells: [
        { rank_id: 'rk1', document_type_id: 'd1', certificate_type_id: null },
        { rank_id: 'rk2', document_type_id: null, certificate_type_id: 'c1' },
        { rank_id: 'rk2', document_type_id: 'd1', certificate_type_id: null },
      ],
    })
  })

  it('selects a whole row with the row checkbox', async () => {
    renderApp(<MatrixEditor matrix={matrix} />)
    await userEvent.click(screen.getByLabelText('Require all items for Master'))
    expect(screen.getByLabelText('Master: BST (A-VI/1)')).toBeChecked()
    expect(screen.getByLabelText('Master: Passport')).toBeChecked()
  })
})
