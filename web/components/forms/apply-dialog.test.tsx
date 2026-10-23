import { act, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import { ApplyButton, ApplyProvider } from '@/components/forms/ApplyDialog'

const FORM = {
  vacancySlug: 'master-ahts',
  ranks: [{ id: 'rank-1', name: 'Master' }],
  defaultRankId: 'rank-1',
  turnstileSiteKey: '',
  apiUrl: 'https://api.example.com',
}

function renderPage() {
  return render(
    <ApplyProvider vacancyTitle="Master AHTS" vacancyId="ABC234567" form={FORM}>
      <ApplyButton />
      <p>Job description</p>
      <ApplyButton />
    </ApplyProvider>,
  )
}

function applicationDialog(): HTMLDialogElement {
  const element = screen.getByRole('heading', { name: 'Apply for Master AHTS' }).closest('dialog')
  if (!element) {
    throw new Error('application dialog not rendered')
  }
  return element
}

afterEach(() => {
  window.history.replaceState(null, '', '/')
})

describe('apply dialog', () => {
  it('opens the application form from either Apply Now button', async () => {
    const user = userEvent.setup()
    renderPage()
    expect(screen.queryByRole('form', { name: 'Job application' })).not.toBeInTheDocument()
    const buttons = screen.getAllByRole('button', { name: 'Apply Now' })
    expect(buttons).toHaveLength(2)
    await user.click(buttons[1]!)
    expect(screen.getByRole('form', { name: 'Job application' })).toBeInTheDocument()
    expect(applicationDialog()).toHaveAttribute('open')
  })

  it('ignores Escape, cancel requests and forced closes so typed answers are not lost', async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(screen.getAllByRole('button', { name: 'Apply Now' })[0]!)
    await user.type(screen.getByLabelText(/Full name/), 'Budi Santoso')
    const dialog = applicationDialog()

    expect(fireEvent.keyDown(dialog, { key: 'Escape' })).toBe(false)
    expect(fireEvent(dialog, new Event('cancel', { cancelable: true }))).toBe(false)
    // A browser can still force a modal shut (for example after repeated Escape presses); it must reopen.
    act(() => {
      dialog.close()
    })

    expect(dialog).toHaveAttribute('open')
    expect(screen.getByLabelText(/Full name/)).toHaveValue('Budi Santoso')
  })

  it('asks for confirmation on the close button and keeps the answers when the user keeps editing', async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(screen.getAllByRole('button', { name: 'Apply Now' })[0]!)
    await user.type(screen.getByLabelText(/Full name/), 'Budi Santoso')

    await user.click(screen.getByRole('button', { name: 'Close application form' }))
    expect(screen.getByRole('alertdialog', { name: 'Close the application form?' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Keep editing' })).toHaveFocus()

    await user.click(screen.getByRole('button', { name: 'Keep editing' }))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(applicationDialog()).toHaveAttribute('open')
    expect(screen.getByLabelText(/Full name/)).toHaveValue('Budi Santoso')
  })

  it('treats Escape on the confirmation as keep editing', async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(screen.getAllByRole('button', { name: 'Apply Now' })[0]!)
    await user.click(screen.getByRole('button', { name: 'Close application form' }))
    const confirmation = screen.getByRole('alertdialog')

    fireEvent(confirmation, new Event('cancel', { cancelable: true }))

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(applicationDialog()).toHaveAttribute('open')
  })

  it('closes and clears the form only after the user confirms the discard', async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(screen.getAllByRole('button', { name: 'Apply Now' })[0]!)
    await user.type(screen.getByLabelText(/Full name/), 'Budi Santoso')

    await user.click(screen.getByRole('button', { name: 'Close application form' }))
    await user.click(screen.getByRole('button', { name: 'Discard and close' }))

    expect(screen.queryByRole('form', { name: 'Job application' })).not.toBeInTheDocument()
    await user.click(screen.getAllByRole('button', { name: 'Apply Now' })[0]!)
    expect(screen.getByLabelText(/Full name/)).toHaveValue('')
  })

  it('opens straight away for links that end in #apply and tidies the address', async () => {
    window.history.replaceState(null, '', '/career/master-ahts#apply')
    renderPage()
    expect(await screen.findByRole('form', { name: 'Job application' })).toBeInTheDocument()
    expect(window.location.hash).toBe('')
  })
})
