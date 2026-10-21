import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { StatusActionDialog } from './StatusActionDialog'

function setup(action: 'blacklist' | 'ready') {
  const onConfirm = vi.fn()
  render(<StatusActionDialog action={action} loading={false} onConfirm={onConfirm} onClose={vi.fn()} />)
  return onConfirm
}

describe('StatusActionDialog', () => {
  it('requires a reason before blacklisting', async () => {
    const user = userEvent.setup()
    const onConfirm = setup('blacklist')
    await user.click(screen.getByRole('button', { name: 'Blacklist' }))
    expect(await screen.findByText('A reason is required.')).toBeInTheDocument()
    expect(onConfirm).not.toHaveBeenCalled()

    await user.type(screen.getByLabelText(/Reason/), '  Falsified documents ')
    await user.click(screen.getByRole('button', { name: 'Blacklist' }))
    expect(onConfirm).toHaveBeenCalledWith('blacklist', 'Falsified documents')
  })

  it('does not require a reason to mark a crew member ready', async () => {
    const user = userEvent.setup()
    const onConfirm = setup('ready')
    await user.click(screen.getByRole('button', { name: 'Mark ready' }))
    expect(onConfirm).toHaveBeenCalledWith('ready', '')
  })
})
