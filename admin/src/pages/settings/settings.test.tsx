import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { SettingGroup } from '@/api/settings'
import { mockApi, renderApp } from '@/test/utils'
import { RecoveryCodesList, RecoveryCodesPanel } from './RecoveryCodes'
import { SettingFieldInput, supportedTimezones } from './SettingFieldInput'
import { SettingsGroupForm } from './SettingsGroupForm'
import { buildSettingsSchema, toFormState, toPayload } from './settingsForm'

afterEach(() => {
  vi.unstubAllGlobals()
})

const emailGroup: SettingGroup = {
  name: 'email',
  fields: [
    { key: 'brevo_api_key', kind: 'text', secret: true, default: '' },
    { key: 'sender_name', kind: 'text', secret: false, default: '' },
    { key: 'admin_notification_emails', kind: 'emails', secret: false, default: [] },
    { key: 'retention', kind: 'int', secret: false, default: 30, min: 1, max: 365 },
  ],
}

describe('settings field rendering by kind', () => {
  const render = (field: SettingGroup['fields'][number], value: string | boolean | string[]) =>
    renderApp(<SettingFieldInput field={field} value={value} onChange={() => undefined} />)

  it('renders a checkbox for bool', () => {
    render({ key: 'maintenance_mode', kind: 'bool', secret: false, default: false }, true)
    expect(screen.getByRole('checkbox', { name: /maintenance mode/i })).toBeChecked()
  })

  it('renders a number input with min and max for int', () => {
    render({ key: 'presign_minutes', kind: 'int', secret: false, default: 5, min: 1, max: 60 }, '5')
    const input = screen.getByLabelText(/presigned url lifetime/i)
    expect(input).toHaveAttribute('type', 'number')
    expect(input).toHaveAttribute('min', '1')
    expect(input).toHaveAttribute('max', '60')
  })

  it('renders a select with the schema options', () => {
    render({ key: 'date_format', kind: 'select', secret: false, default: '', options: ['DD-MMM-YYYY', 'YYYY-MM-DD'] }, 'YYYY-MM-DD')
    const select = screen.getByLabelText('Date Format')
    expect(select).toHaveValue('YYYY-MM-DD')
    expect(within(select).getAllByRole('option')).toHaveLength(2)
  })

  it('renders a tag list for emails and validates entries', async () => {
    const onChange = vi.fn()
    renderApp(<SettingFieldInput field={{ key: 'admin_notification_emails', kind: 'emails', secret: false, default: [] }} value={['a@x.com']} onChange={onChange} />)
    expect(screen.getByText('a@x.com')).toBeInTheDocument()
    const input = screen.getByLabelText('Admin notification addresses')
    await userEvent.type(input, 'not-an-email{Enter}')
    expect(await screen.findByText('Enter a valid email address.')).toBeInTheDocument()
    await userEvent.clear(input)
    await userEvent.type(input, 'b@x.com{Enter}')
    expect(onChange).toHaveBeenCalledWith(['a@x.com', 'b@x.com'])
  })

  it('renders a time input and a color picker with hex text', () => {
    render({ key: 'run_time', kind: 'time', secret: false, default: '02:00' }, '02:00')
    expect(screen.getByLabelText(/run time/i)).toHaveAttribute('type', 'time')
    render({ key: 'primary_color', kind: 'color', secret: false, default: '#0B2545' }, '#0B2545')
    expect(screen.getByLabelText('Primary Color picker')).toBeInTheDocument()
  })

  it('renders the home sections as an ordered toggle list', async () => {
    const onChange = vi.fn()
    renderApp(
      <SettingFieldInput
        field={{ key: 'home_sections', kind: 'list', secret: false, default: [], options: ['hero', 'news', 'about'] }}
        value={['hero', 'news']}
        onChange={onChange}
      />,
    )
    expect(screen.getByRole('checkbox', { name: 'Show About' })).not.toBeChecked()
    await userEvent.click(screen.getByRole('button', { name: 'Move Hero down' }))
    expect(onChange).toHaveBeenCalledWith(['news', 'hero'])
    await userEvent.click(screen.getByRole('checkbox', { name: 'Show About' }))
    expect(onChange).toHaveBeenLastCalledWith(['hero', 'news', 'about'])
  })

  it('offers the current timezone even when the runtime does not list it', () => {
    expect(supportedTimezones('Mars/Olympus')[0]).toBe('Mars/Olympus')
    expect(supportedTimezones('Asia/Jakarta')).toContain('Asia/Jakarta')
  })
})

describe('secret settings', () => {
  it('shows the mask and never a stored value, and keeps the secret when left empty', () => {
    const values = { brevo_api_key: { configured: true, masked: '••••1234' }, sender_name: 'Crew', admin_notification_emails: ['a@x.com'], retention: 30 }
    const state = toFormState(emailGroup, values)
    expect(state.brevo_api_key).toBe('')
    expect(toPayload(emailGroup, state).brevo_api_key).toBe('')
    expect(toPayload(emailGroup, { ...state, brevo_api_key: ' new-key ' }).brevo_api_key).toBe('new-key')
  })

  it('renders configured status with the mask and a replace input that is empty', () => {
    renderApp(<SettingFieldInput field={emailGroup.fields[0]!} value="" onChange={() => undefined} secret={{ configured: true, masked: '••••1234' }} />)
    expect(screen.getByText('••••1234')).toBeInTheDocument()
    const input = screen.getByLabelText(/replace brevo api key/i)
    expect(input).toHaveValue('')
    expect(input).toHaveAttribute('type', 'password')
    expect(screen.getByText('Leave empty to keep the current value.')).toBeInTheDocument()
  })

  it('sends an empty string for an untouched secret when saving another field', async () => {
    const calls = mockApi((c) => {
      if (c.method === 'PUT') return { body: { data: { brevo_api_key: { configured: true, masked: '••••1234' }, sender_name: 'Renamed', admin_notification_emails: ['a@x.com'], retention: 30 } } }
      return { body: { data: { brevo_api_key: { configured: true, masked: '••••1234' }, sender_name: 'Crew', admin_notification_emails: ['a@x.com'], retention: 30 } } }
    })
    renderApp(<SettingsGroupForm group={emailGroup} onDirtyChange={() => undefined} />)
    const name = await screen.findByLabelText('Sender Name')
    await userEvent.clear(name)
    await userEvent.type(name, 'Renamed')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    const put = calls.find((c) => c.method === 'PUT')
    expect(put?.path).toBe('/settings/email')
    expect(put?.body).toMatchObject({ brevo_api_key: '', sender_name: 'Renamed' })
  })

  it('blocks saving an out-of-range integer', async () => {
    const calls = mockApi(() => ({ body: { data: { brevo_api_key: { configured: false, masked: '' }, sender_name: 'Crew', admin_notification_emails: [], retention: 30 } } }))
    renderApp(<SettingsGroupForm group={emailGroup} onDirtyChange={() => undefined} />)
    const retention = await screen.findByLabelText('Retention')
    await userEvent.clear(retention)
    await userEvent.type(retention, '999')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText('Enter a whole number between 1 and 365.')).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'PUT')).toBe(false)
  })

  it('schema validation mirrors server rules for email lists', () => {
    const schema = buildSettingsSchema(emailGroup)
    const bad = schema.safeParse({ brevo_api_key: '', sender_name: 'x', admin_notification_emails: ['nope'], retention: '5' })
    expect(bad.success).toBe(false)
  })
})

describe('recovery codes', () => {
  const codes = Array.from({ length: 10 }, (_, i) => `CODE-000${i}-AAAA`)

  it('shows all ten codes once with copy and download actions', () => {
    renderApp(<RecoveryCodesList codes={codes} onDone={() => undefined} />)
    expect(within(screen.getByRole('list', { name: 'Recovery codes' })).getAllByRole('listitem')).toHaveLength(10)
    expect(screen.getByRole('button', { name: 'Copy' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Download .txt' })).toBeInTheDocument()
    expect(screen.getByText(/shown only once/)).toBeInTheDocument()
  })

  it('requires confirmation before regenerating', async () => {
    const calls = mockApi(() => undefined)
    renderApp(<RecoveryCodesPanel />)
    await userEvent.click(screen.getByRole('button', { name: 'Regenerate recovery codes' }))
    expect(calls).toHaveLength(0)
  })

  it('shows the new codes after the admin confirms', async () => {
    mockApi(() => ({ body: { data: { codes } } }))
    renderApp(<RecoveryCodesPanel />)
    await userEvent.click(screen.getByRole('button', { name: 'Regenerate recovery codes' }))
    await userEvent.click(screen.getByRole('button', { name: 'Regenerate' }))
    expect(await screen.findByRole('list', { name: 'Recovery codes' })).toBeInTheDocument()
    expect(screen.getByText('CODE-0007-AAAA')).toBeInTheDocument()
  })
})
