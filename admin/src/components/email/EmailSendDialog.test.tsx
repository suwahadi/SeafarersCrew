import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { ToastProvider } from '../ui/Toast'
import { EmailSendDialog } from './EmailSendDialog'

const send = vi.fn<(input: unknown) => Promise<unknown>>()

beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function showModal(this: HTMLDialogElement) {
    this.setAttribute('open', '')
  }
  HTMLDialogElement.prototype.close = function close(this: HTMLDialogElement) {
    this.removeAttribute('open')
  }
})

vi.mock('@/api/email', () => ({
  emailApi: {
    templates: () =>
      Promise.resolve([
        { code: 'send_contract', name: 'Send contract', subject: 's', updated_at: '' },
        { code: 'send_quotation', name: 'Send quotation', subject: 's', updated_at: '' },
      ]),
    template: (code: string) => Promise.resolve({ code, name: code, subject: 'Your contract {contract_id}', body_html: '<p>Dear {crew_name}</p>', updated_at: '' }),
    send: (input: unknown) => send(input),
  },
}))

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={qc}>
      <ToastProvider>{children}</ToastProvider>
    </QueryClientProvider>
  )
}

function setup() {
  const onClose = vi.fn()
  render(
    <EmailSendDialog
      open
      onClose={onClose}
      relatedType="contract"
      relatedId="ABCDEFGH1"
      attachments={[{ fileId: '11111111-1111-1111-1111-111111111111', name: 'Contract.pdf' }]}
    />,
    { wrapper },
  )
  return { onClose }
}

describe('EmailSendDialog', () => {
  beforeEach(() => send.mockReset())

  it('prefills subject and body from the template for the related type', async () => {
    setup()
    expect(await screen.findByDisplayValue('Your contract {contract_id}')).toBeInTheDocument()
    expect(screen.getByDisplayValue('<p>Dear {crew_name}</p>')).toBeInTheDocument()
  })

  it('blocks sending to an invalid address', async () => {
    const user = userEvent.setup()
    setup()
    await screen.findByDisplayValue('Your contract {contract_id}')
    await user.type(screen.getByLabelText(/^To/), 'not-an-email')
    await user.click(screen.getByRole('button', { name: 'Send email' }))
    expect(await screen.findByText('Invalid address: not-an-email')).toBeInTheDocument()
    expect(send).not.toHaveBeenCalled()
  })

  it('queues the email with parsed recipients, attachments and the related record', async () => {
    const user = userEvent.setup()
    send.mockResolvedValue({ log: {}, job_id: 'j' })
    const { onClose } = setup()
    await screen.findByDisplayValue('Your contract {contract_id}')
    await user.type(screen.getByLabelText(/^To/), 'Crew@Example.com, ops@example.com')
    await user.type(screen.getByLabelText('CC'), 'boss@example.com')
    await user.click(screen.getByRole('button', { name: 'Send email' }))
    await waitFor(() => expect(send).toHaveBeenCalled())
    expect(send).toHaveBeenCalledWith(
      expect.objectContaining({
        template_code: 'send_contract',
        to: ['crew@example.com', 'ops@example.com'],
        cc: ['boss@example.com'],
        related_type: 'contract',
        related_id: 'ABCDEFGH1',
        attachment_file_ids: ['11111111-1111-1111-1111-111111111111'],
      }),
    )
    expect(await screen.findByText(/Email queued/)).toBeInTheDocument()
    expect(onClose).toHaveBeenCalled()
  })
})
