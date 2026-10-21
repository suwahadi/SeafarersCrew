import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SESSION_KEY } from '@/app/auth'
import { mockApi, renderApp } from '@/test/utils'
import { PasswordCard } from './AccountPage'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('change password', () => {
  it('rejects short and mismatched passwords before calling the API', async () => {
    const calls = mockApi(() => undefined)
    renderApp(<PasswordCard />)
    const user = userEvent.setup()
    await user.type(screen.getByLabelText(/^Current password/), 'old-password')
    await user.type(screen.getByLabelText(/^New password/), 'short')
    await user.type(screen.getByLabelText(/^Confirm new password/), 'other')
    await user.click(screen.getByRole('button', { name: 'Change password' }))
    expect(await screen.findByText(/at least 12 characters/)).toBeInTheDocument()
    expect(screen.getByText('The passwords do not match.')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('stores the returned session after a successful change', async () => {
    const session = { admin: { id: '1', name: 'Admin', email: 'a@x.com', must_change_password: false, password_changed_at: '' }, csrf_token: 'tok' }
    const calls = mockApi(() => ({ body: { data: session } }))
    const { client } = renderApp(<PasswordCard />)
    const user = userEvent.setup()
    await user.type(screen.getByLabelText(/^Current password/), 'old-password')
    await user.type(screen.getByLabelText(/^New password/), 'a-much-longer-password')
    await user.type(screen.getByLabelText(/^Confirm new password/), 'a-much-longer-password')
    await user.click(screen.getByRole('button', { name: 'Change password' }))
    await waitFor(() => expect(client.getQueryData(SESSION_KEY)).toEqual(session))
    expect(calls[0]).toMatchObject({ method: 'PUT', path: '/me/password', body: { current_password: 'old-password', new_password: 'a-much-longer-password' } })
  })
})
