import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { StageMoveDialog } from './StageMoveDialog'

function setup(to: 'rejected' | 'screening') {
  const onConfirm = vi.fn()
  const onClose = vi.fn()
  render(<StageMoveDialog open applicantName="Budi Santoso" from="new" to={to} onConfirm={onConfirm} onClose={onClose} />)
  return { onConfirm, onClose }
}

describe('StageMoveDialog', () => {
  it('requires a note when rejecting an applicant', async () => {
    const user = userEvent.setup()
    const { onConfirm } = setup('rejected')
    await user.click(screen.getByRole('button', { name: 'Move applicant' }))
    expect(screen.getByRole('alert')).toHaveTextContent('A note is required when rejecting an applicant.')
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('confirms a rejection with the trimmed note', async () => {
    const user = userEvent.setup()
    const { onConfirm } = setup('rejected')
    await user.type(screen.getByLabelText(/^Note/), '  Documents expired  ')
    await user.click(screen.getByRole('button', { name: 'Move applicant' }))
    expect(onConfirm).toHaveBeenCalledWith('Documents expired')
  })

  it('allows moving to another stage without a note', async () => {
    const user = userEvent.setup()
    const { onConfirm } = setup('screening')
    await user.click(screen.getByRole('button', { name: 'Move applicant' }))
    expect(onConfirm).toHaveBeenCalledWith('')
  })

  it('shows an error returned by the server', () => {
    render(<StageMoveDialog open applicantName="Budi" from="new" to="screening" error="The applicant was moved by someone else." onConfirm={vi.fn()} onClose={vi.fn()} />)
    expect(screen.getByRole('alert')).toHaveTextContent('The applicant was moved by someone else.')
  })

  it('moves focus to the note field when it opens', () => {
    setup('screening')
    expect(screen.getByLabelText(/^Note/)).toHaveFocus()
  })
})
