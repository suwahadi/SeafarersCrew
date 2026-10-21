import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ContactMessage } from '@/api/contactMessages'
import { ToastProvider } from '@/components/ui/Toast'
import ContactInboxPage from './ContactInboxPage'

const patch = vi.fn()
const list = vi.fn()

const unread: ContactMessage = {
  id: 'm1',
  name: 'Ana Putri',
  email: 'ana@example.com',
  phone: '+62 811',
  subject: 'Crew inquiry',
  message: 'Line one\nLine two <b>not bold</b>',
  ip_address: '203.0.113.9',
  is_read: false,
  is_archived: false,
  created_at: '2026-10-01T03:00:00Z',
}

vi.mock('@/api/contactMessages', () => ({
  UNREAD_COUNT_KEY: ['contact-messages', 'unread-count'],
  contactMessagesApi: {
    list: (...args: unknown[]) => list(...args) as unknown,
    patch: (...args: unknown[]) => patch(...args) as unknown,
  },
}))

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <ToastProvider>
        <ContactInboxPage />
      </ToastProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  patch.mockReset()
  list.mockReset()
  list.mockResolvedValue({ data: [unread], meta: { page: 1, per_page: 20, total: 1 } })
})

describe('ContactInboxPage', () => {
  it('marks an unread message as read when it is opened', async () => {
    const user = userEvent.setup()
    patch.mockResolvedValue({ ...unread, is_read: true })
    renderPage()
    await user.click(await screen.findByText('Ana Putri'))
    await waitFor(() => expect(patch).toHaveBeenCalledWith('m1', { is_read: true }))
    expect(await screen.findByRole('button', { name: 'Mark as unread' })).toBeInTheDocument()
  })

  it('shows the message as plain text without interpreting markup', async () => {
    const user = userEvent.setup()
    patch.mockResolvedValue({ ...unread, is_read: true })
    renderPage()
    await user.click(await screen.findByText('Ana Putri'))
    const detail = await screen.findByRole('article', { name: 'Message detail' })
    expect(detail).toHaveTextContent('Line two <b>not bold</b>')
    expect(detail.querySelector('b')).toBeNull()
  })

  it('does not patch a message that is already read when it is opened', async () => {
    const user = userEvent.setup()
    list.mockResolvedValue({ data: [{ ...unread, is_read: true }], meta: { page: 1, per_page: 20, total: 1 } })
    renderPage()
    await user.click(await screen.findByText('Ana Putri'))
    expect(await screen.findByRole('button', { name: 'Mark as unread' })).toBeInTheDocument()
    expect(patch).not.toHaveBeenCalled()
  })

  it('archives the open message', async () => {
    const user = userEvent.setup()
    patch.mockImplementation((_id: string, body: { is_read?: boolean; is_archived?: boolean }) => Promise.resolve({ ...unread, is_read: true, ...body }))
    renderPage()
    await user.click(await screen.findByText('Ana Putri'))
    await user.click(await screen.findByRole('button', { name: 'Archive' }))
    await waitFor(() => expect(patch).toHaveBeenCalledWith('m1', { is_archived: true }))
    expect(await screen.findByRole('button', { name: 'Unarchive' })).toBeInTheDocument()
  })

  it('offers a mailto reply to the sender', async () => {
    const user = userEvent.setup()
    patch.mockResolvedValue({ ...unread, is_read: true })
    renderPage()
    await user.click(await screen.findByText('Ana Putri'))
    const reply = await screen.findByRole('link', { name: 'Reply by email' })
    expect(reply).toHaveAttribute('href', 'mailto:ana@example.com?subject=Re%3A%20Crew%20inquiry')
  })

  it('requests the messages of the selected status tab', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('Ana Putri')
    await user.click(screen.getByRole('tab', { name: 'Archived' }))
    await waitFor(() => expect(list).toHaveBeenCalledWith(expect.objectContaining({ status: 'archived' }), expect.anything()))
  })
})
