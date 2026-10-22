import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { Language, Stat, Testimonial } from '@/api/cms'
import { mockApi, renderApp, type RecordedCall } from '@/test/utils'
import { BulkUploader } from './GalleryBulkUpload'
import { CmsListPage } from './CmsListPage'
import { languageConfig } from './LanguagesPage'
import { statConfig } from './StatsPage'
import { TestimonialForm } from './TestimonialsPage'

const upload = vi.fn()

vi.mock('@/api/files', () => ({ filesApi: { upload: (...args: unknown[]) => upload(...args) as unknown } }))

afterEach(() => {
  vi.unstubAllGlobals()
  upload.mockReset()
})

const base = { is_active: true, sort_order: 0, created_at: '', updated_at: '' }

function stat(id: string, label: string): Stat {
  return { ...base, id, number: 1200, suffix: '+', label }
}

function listReply(rows: unknown[]) {
  return { body: { data: rows, meta: { page: 1, per_page: 100, total: rows.length } } }
}

describe('reordering a list', () => {
  const rows = [stat('s1', 'Seafarers Deployed'), stat('s2', 'Vessels Served'), stat('s3', 'Countries')]

  it('shows the list formatted and sends ids in the new order when a row is moved down with the keyboard', async () => {
    const calls = mockApi((c) => (c.method === 'GET' ? listReply(rows) : { status: 204 }))
    renderApp(<CmsListPage config={statConfig} />)
    expect(await screen.findByText('1,200+ Seafarers Deployed')).toBeInTheDocument()
    screen.getByRole('button', { name: /^Reorder Seafarers Deployed\./ }).focus()
    await userEvent.keyboard('{ArrowDown}')
    await waitFor(() => expect(calls.some((c) => c.method === 'PATCH')).toBe(true))
    const patch = calls.find((c) => c.method === 'PATCH')
    expect(patch?.path).toBe('/stats/reorder')
    expect(patch?.body).toEqual({ ids: ['s2', 's1', 's3'] })
  })

  it('sends the new order when a row handle is dragged onto another', async () => {
    const calls = mockApi((c) => (c.method === 'GET' ? listReply(rows) : { status: 204 }))
    renderApp(<CmsListPage config={statConfig} />)
    await screen.findByText('1,200+ Seafarers Deployed')
    const items = screen.getAllByRole('listitem')
    const handle = screen.getByRole('button', { name: /^Reorder Seafarers Deployed\./ })
    if (!items[2]) throw new Error('missing row')
    Object.assign(handle, { setPointerCapture: () => undefined, releasePointerCapture: () => undefined })
    document.elementsFromPoint = () => [items[2] as Element]
    fireEvent.pointerDown(handle, { pointerId: 1, button: 0, pointerType: 'touch' })
    fireEvent.pointerMove(handle, { pointerId: 1, clientX: 10, clientY: 200 })
    fireEvent.pointerUp(handle, { pointerId: 1, clientX: 10, clientY: 200 })
    await waitFor(() => expect(calls.some((c) => c.method === 'PATCH')).toBe(true))
    expect(calls.find((c) => c.method === 'PATCH')?.body).toEqual({ ids: ['s2', 's3', 's1'] })
  })

  it('rolls the order back when saving fails', async () => {
    mockApi((c) => {
      if (c.method === 'GET') return listReply(rows)
      return { status: 500, body: { error: { code: 'server_error', message: 'Could not save the order.' } } }
    })
    renderApp(<CmsListPage config={statConfig} />)
    await screen.findByText('1,200+ Seafarers Deployed')
    screen.getByRole('button', { name: /^Reorder Seafarers Deployed\./ }).focus()
    await userEvent.keyboard('{ArrowDown}')
    expect(await screen.findByText('Could not save the order.')).toBeInTheDocument()
    await waitFor(() => {
      const first = screen.getAllByRole('listitem')[0]
      expect(first).toHaveTextContent('Seafarers Deployed')
    })
  })

  it('toggles active with the full write payload', async () => {
    const calls: RecordedCall[] = mockApi((c) => (c.method === 'GET' ? listReply(rows) : { body: { data: { ...rows[0], is_active: false } } }))
    renderApp(<CmsListPage config={statConfig} />)
    await userEvent.click(await screen.findByRole('switch', { name: 'Seafarers Deployed: active' }))
    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    const put = calls.find((c) => c.method === 'PUT')
    expect(put?.path).toBe('/stats/s1')
    expect(put?.body).toEqual({ number: 1200, suffix: '+', label: 'Seafarers Deployed', is_active: false })
  })
})

describe('source language', () => {
  const english: Language = { ...base, id: 'l1', name: 'English', code: 'en', flag: null, flag_code: 'gb', is_source: true }
  const indonesian: Language = { ...base, id: 'l2', name: 'Indonesian', code: 'id', flag: null, flag_code: 'id', is_source: false }

  it('disables deactivating and deleting the source language but not the others', async () => {
    mockApi(() => listReply([english, indonesian]))
    renderApp(<CmsListPage config={languageConfig} />)
    const toggle = await screen.findByRole('switch', { name: 'English: active' })
    expect(toggle).toBeDisabled()
    expect(toggle).toHaveAttribute('title', expect.stringContaining('source language'))
    expect(screen.getByRole('button', { name: 'Delete English' })).toBeDisabled()
    expect(screen.getByText('Source')).toBeInTheDocument()
    expect(screen.getByRole('switch', { name: 'Indonesian: active' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Delete Indonesian' })).toBeEnabled()
  })

  it('shows the flag code as a text badge when there is no uploaded flag', async () => {
    mockApi(() => listReply([indonesian]))
    renderApp(<CmsListPage config={languageConfig} />)
    const row = (await screen.findAllByRole('listitem'))[0]
    expect(row && within(row).getByText('id', { selector: 'span.uppercase' })).toBeInTheDocument()
  })
})

describe('testimonial form', () => {
  it('requires a rating and shows a keyboard accessible radio group', async () => {
    const calls = mockApi(() => undefined)
    renderApp(<TestimonialForm row={null} onDone={() => undefined} />)
    await userEvent.type(screen.getByLabelText(/^Name/), 'Capt. Rizal')
    await userEvent.type(screen.getByLabelText(/^Quote/), 'Great crew agency.')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Choose a rating from 1 to 5.')).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'POST')).toBe(false)
    expect(screen.getByRole('radiogroup', { name: /Rating/ })).toBeInTheDocument()
    expect(screen.getAllByRole('radio')).toHaveLength(5)
  })

  it('posts the chosen rating and counts quote characters', async () => {
    const calls = mockApi((c) => (c.method === 'POST' ? { status: 201, body: { data: { id: 't1' } } } : { body: { data: [], meta: {} } }))
    const onDone = vi.fn()
    renderApp(<TestimonialForm row={null} onDone={onDone} />)
    await userEvent.type(screen.getByLabelText(/^Name/), 'Capt. Rizal')
    await userEvent.type(screen.getByLabelText(/^Quote/), 'Great')
    expect(screen.getByText('5 / 1000')).toBeInTheDocument()
    await userEvent.click(screen.getByLabelText('4 stars'))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(onDone).toHaveBeenCalled())
    expect(calls.find((c) => c.method === 'POST')?.body).toMatchObject({ name: 'Capt. Rizal', quote: 'Great', rating: 4, photo_file_id: null, is_active: true })
  })

  it('preselects the saved rating when editing', () => {
    mockApi(() => undefined)
    const row: Testimonial = { ...base, id: 't1', name: 'A', position: '', photo: null, quote: 'q', rating: 3 }
    renderApp(<TestimonialForm row={row} onDone={() => undefined} />)
    expect(screen.getByLabelText('3 stars')).toBeChecked()
  })
})

describe('bulk upload', () => {
  function pick(count: number): File[] {
    return Array.from({ length: count }, (_, i) => new File(['x'], `crew-photo_${i + 1}.jpg`, { type: 'image/jpeg' }))
  }

  it('uploads each file as a CMS image, prefilling alt text from the file name, then posts one bulk request', async () => {
    upload.mockImplementation((file: File) => Promise.resolve({ id: `id-${file.name}` }))
    const calls = mockApi((c) => (c.path === '/gallery/bulk' ? { status: 201, body: { data: [] } } : undefined))
    const onDone = vi.fn()
    renderApp(<BulkUploader onDone={onDone} />)
    await userEvent.upload(screen.getByLabelText('Choose images'), pick(2))
    const alt = await screen.findByLabelText('Alt text for crew-photo_1.jpg')
    expect(alt).toHaveValue('Crew photo 1')
    expect(upload).toHaveBeenCalledWith(expect.any(File), 'cms_image')
    await userEvent.type(screen.getByLabelText('Caption for crew-photo_1.jpg'), ' On deck ')
    await userEvent.click(screen.getByLabelText('Show all on home page'))
    const add = screen.getByRole('button', { name: 'Add 2 photos' })
    await waitFor(() => expect(add).toBeEnabled())
    await userEvent.click(add)
    await waitFor(() => expect(onDone).toHaveBeenCalled())
    const posts = calls.filter((c) => c.path === '/gallery/bulk')
    expect(posts).toHaveLength(1)
    expect(posts[0]?.body).toEqual({
      items: [
        { image_file_id: 'id-crew-photo_1.jpg', alt_text: 'Crew photo 1', caption: 'On deck' },
        { image_file_id: 'id-crew-photo_2.jpg', alt_text: 'Crew photo 2', caption: '' },
      ],
      show_on_home: true,
      is_active: true,
    })
  })

  it('splits more than 50 images into several bulk requests', async () => {
    upload.mockImplementation((file: File) => Promise.resolve({ id: `id-${file.name}` }))
    const calls = mockApi((c) => (c.path === '/gallery/bulk' ? { status: 201, body: { data: [] } } : undefined))
    const onDone = vi.fn()
    renderApp(<BulkUploader onDone={onDone} />)
    await userEvent.upload(screen.getByLabelText('Choose images'), pick(53))
    const add = await screen.findByRole('button', { name: 'Add 53 photos' })
    await waitFor(() => expect(add).toBeEnabled(), { timeout: 5000 })
    await userEvent.click(add)
    await waitFor(() => expect(onDone).toHaveBeenCalled())
    const sizes = calls.filter((c) => c.path === '/gallery/bulk').map((c) => (c.body as { items: unknown[] }).items.length)
    expect(sizes).toEqual([50, 3])
  })

  it('keeps failed uploads out of the request and lets them be retried', async () => {
    upload.mockRejectedValueOnce(new Error('The image could not be processed.')).mockResolvedValue({ id: 'ok' })
    mockApi(() => undefined)
    renderApp(<BulkUploader onDone={() => undefined} />)
    await userEvent.upload(screen.getByLabelText('Choose images'), pick(1))
    expect(await screen.findByText('The image could not be processed.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add 0 photos' })).toBeDisabled()
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Add 1 photo' })).toBeEnabled())
  })
})
