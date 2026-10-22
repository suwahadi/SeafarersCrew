import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApplicationForm } from '@/components/forms/ApplicationForm'
import { ContactForm } from '@/components/forms/ContactForm'
import { MapEmbed } from '@/components/ui/map-embed'

vi.mock('next/link', () => ({
  default: ({ href, children, ...rest }: { href: string; children: ReactNode }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))

const RANKS = [
  { id: 'rank-1', name: 'Master' },
  { id: 'rank-2', name: 'Chief Officer' },
]

function jsonResponse(status: number, body: unknown, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json', ...headers } })
}

function renderApplication(siteKey = '') {
  return render(
    <ApplicationForm vacancySlug="master-ahts" ranks={RANKS} defaultRankId="rank-1" turnstileSiteKey={siteKey} apiUrl="http://api.test" />,
  )
}

async function fillApplication(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/Full name/), 'Budi Santoso')
  await user.type(screen.getByLabelText(/Email address/), 'budi@example.com')
  await user.type(screen.getByLabelText(/Phone number/), '+628123456789')
  await user.type(screen.getByLabelText(/Date of birth/), '1990-05-17')
  await user.type(screen.getByLabelText(/Nationality/), 'Indonesian')
  await user.type(screen.getByLabelText(/Years of sea experience/), '8')
  await user.upload(screen.getByLabelText(/^CV/), new File(['%PDF'], 'cv.pdf', { type: 'application/pdf' }))
  await user.click(screen.getByLabelText(/I have read the/))
}

describe('ApplicationForm', () => {
  const fetchMock = vi.fn<typeof fetch>()

  beforeEach(() => {
    fetchMock.mockReset()
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.getElementById('turnstile-api')?.remove()
  })

  it('shows announced errors and focuses the first invalid field on an empty submit', async () => {
    const user = userEvent.setup()
    renderApplication()
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    const name = screen.getByLabelText(/Full name/)
    expect(name).toHaveFocus()
    expect(name).toHaveAttribute('aria-invalid', 'true')
    expect(name.getAttribute('aria-describedby')).toContain('application-full_name-error')
    expect(screen.getByText('Enter your full name.')).toHaveAttribute('role', 'alert')
    expect(screen.getByText('Upload your CV.')).toBeInTheDocument()
    expect(screen.getByText('You must accept the Privacy Policy to apply.')).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('preselects the vacancy rank and links the privacy policy', () => {
    renderApplication()
    expect(screen.getByLabelText(/Rank applied for/)).toHaveValue('rank-1')
    expect(screen.getByRole('link', { name: 'Privacy Policy' })).toHaveAttribute('href', '/privacy-policy')
  })

  it('hides the honeypot from users and assistive technology', () => {
    renderApplication()
    const honeypot = document.querySelector('input[name="website"]')
    expect(honeypot).toHaveAttribute('tabindex', '-1')
    expect(honeypot).toHaveAttribute('autocomplete', 'off')
    expect(honeypot?.closest('[aria-hidden="true"]')).toHaveClass('hp-field')
  })

  it('adds, removes and limits supporting documents', async () => {
    const user = userEvent.setup({ applyAccept: false })
    renderApplication()
    const input = screen.getByLabelText(/Supporting documents/)
    await user.upload(input, [new File(['a'], 'a.pdf'), new File(['b'], 'b.png')])
    expect(screen.getByText('a.pdf')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Remove a.pdf' }))
    expect(screen.queryByText('a.pdf')).not.toBeInTheDocument()
    expect(screen.getByText('b.png')).toBeInTheDocument()

    await user.upload(input, new File(['x'], 'virus.exe'))
    expect(screen.getByText(/virus.exe: only PDF/)).toBeInTheDocument()
    expect(screen.queryByText('virus.exe')).not.toBeInTheDocument()

    const many = Array.from({ length: 12 }, (_, index) => new File(['x'], `f${String(index)}.pdf`))
    await user.upload(input, many)
    expect(screen.getAllByRole('listitem').filter((item) => item.textContent.includes('Remove')).length).toBe(10)
    expect(screen.getByText('You can upload at most 10 supporting documents.')).toBeInTheDocument()
  })

  it('rejects an oversize CV on selection', async () => {
    const user = userEvent.setup()
    renderApplication()
    const big = new File([new Uint8Array(11 * 1024 * 1024)], 'big.pdf', { type: 'application/pdf' })
    await user.upload(screen.getByLabelText(/^CV/), big)
    expect(screen.getByText(/each file may not be larger than 10 MB/)).toBeInTheDocument()
  })

  it('submits multipart data and shows the application id on success', async () => {
    fetchMock.mockResolvedValue(jsonResponse(201, { data: { application_id: 'ABCD12345' } }))
    const user = userEvent.setup()
    renderApplication()
    await fillApplication(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    expect(await screen.findByTestId('application-id')).toHaveTextContent('ABCD12345')
    expect(screen.getByText(/A confirmation email was sent/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Browse more vacancies' })).toHaveAttribute('href', '/career')
    const [url, init] = fetchMock.mock.calls[0] ?? []
    expect(url).toBe('http://api.test/api/public/v1/applications')
    const body = (init!).body! as FormData
    expect(body.get('vacancy')).toBe('master-ahts')
    expect(body.get('rank_applied_id')).toBe('rank-1')
    expect(body.has('cf-turnstile-response')).toBe(false)
  })

  it('shows a banner for a duplicate application and keeps the form', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(409, { error: { code: 'duplicate_application', message: 'You have already applied for this vacancy.' } }),
    )
    const user = userEvent.setup()
    renderApplication()
    await fillApplication(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent('You have already applied for this vacancy.')
    expect(screen.getByRole('button', { name: 'Submit application' })).toBeEnabled()
  })

  it('maps server field errors onto inputs', async () => {
    fetchMock.mockResolvedValue(jsonResponse(422, { error: { code: 'validation_failed', fields: { email: 'Server says no.' } } }))
    const user = userEvent.setup()
    renderApplication()
    await fillApplication(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    expect(await screen.findByText('Server says no.')).toBeInTheDocument()
    expect(screen.getByLabelText(/Email address/)).toHaveFocus()
  })

  it('shows a friendly message when rate limited and on network failure', async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(429, { error: { code: 'too_many_attempts' } }, { 'Retry-After': '600' }))
    const user = userEvent.setup()
    renderApplication()
    await fillApplication(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    expect(await screen.findByText(/try again in about 10 minutes/)).toBeInTheDocument()

    fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'))
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    expect(await screen.findByText(/Check your connection/)).toBeInTheDocument()
  })

  it('disables the button and shows a spinner while sending', async () => {
    let finish: (response: Response) => void = () => undefined
    fetchMock.mockReturnValue(new Promise<Response>((resolve) => { finish = resolve }))
    const user = userEvent.setup()
    renderApplication()
    await fillApplication(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    const sending = await screen.findByRole('button', { name: 'Sending application' })
    expect(sending).toBeDisabled()
    expect(sending.querySelector('.spinner')).not.toBeNull()
    finish(jsonResponse(201, { data: { application_id: 'ZZZ' } }))
    await waitFor(() => { expect(screen.getByTestId('application-id')).toBeInTheDocument() })
  })

  it('does not load Turnstile when the site key is empty', async () => {
    const user = userEvent.setup()
    renderApplication('')
    await user.click(screen.getByLabelText(/Full name/))
    expect(document.getElementById('turnstile-api')).toBeNull()
    expect(document.querySelector('[data-turnstile]')).toBeNull()
  })

  it('loads Turnstile lazily only after the form is touched', async () => {
    const user = userEvent.setup()
    renderApplication('site-key')
    expect(document.getElementById('turnstile-api')).toBeNull()
    await user.click(screen.getByLabelText(/Full name/))
    await waitFor(() => { expect(document.getElementById('turnstile-api')).not.toBeNull() })
    expect(document.getElementById('turnstile-api')).toHaveAttribute('src', expect.stringContaining('challenges.cloudflare.com/turnstile/v0/api.js'))
  })
})

describe('ContactForm', () => {
  const fetchMock = vi.fn<typeof fetch>()

  beforeEach(() => {
    fetchMock.mockReset()
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('validates required fields and the optional phone format', async () => {
    const user = userEvent.setup()
    render(<ContactForm turnstileSiteKey="" apiUrl="http://api.test" />)
    await user.type(screen.getByLabelText(/Phone number/), '0812')
    await user.click(screen.getByRole('button', { name: 'Send message' }))
    expect(screen.getByLabelText(/Your name/)).toHaveFocus()
    expect(screen.getByText('Enter your message.')).toBeInTheDocument()
    expect(screen.getByText(/international format with country code, for example/)).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('counts message characters live', async () => {
    const user = userEvent.setup()
    render(<ContactForm turnstileSiteKey="" apiUrl="http://api.test" />)
    await user.type(screen.getByLabelText(/Message/), 'Hello')
    expect(screen.getByText('5 / 2000')).toBeInTheDocument()
  })

  it('posts json and shows the thank you panel', async () => {
    fetchMock.mockResolvedValue(jsonResponse(201, { data: { received: true } }))
    const user = userEvent.setup()
    render(<ContactForm turnstileSiteKey="" apiUrl="http://api.test" />)
    await user.type(screen.getByLabelText(/Your name/), 'Budi')
    await user.type(screen.getByLabelText(/Email address/), 'budi@example.com')
    await user.type(screen.getByLabelText(/Subject/), 'Hi')
    await user.type(screen.getByLabelText(/Message/), 'Hello')
    await user.click(screen.getByRole('button', { name: 'Send message' }))
    expect(await screen.findByText('Thank you, we received your message')).toBeInTheDocument()
    const [url, init] = fetchMock.mock.calls[0] ?? []
    expect(url).toBe('http://api.test/api/public/v1/contact')
    expect(JSON.parse((init!).body! as string)).toMatchObject({ name: 'Budi', subject: 'Hi', website: '' })
  })
})

describe('MapEmbed', () => {
  it('shows the map right away with a link to the larger map', () => {
    render(<MapEmbed coordinates={{ lat: -6.2, lng: 106.8 }} />)
    const frame = document.querySelector('iframe')
    expect(frame).toHaveAttribute('loading', 'lazy')
    expect(frame?.getAttribute('src')).toContain('openstreetmap.org/export/embed.html?bbox=')
    expect(frame?.getAttribute('src')).toContain('marker=-6.200000%2C106.800000')
    expect(screen.getByRole('link', { name: 'View larger map' })).toHaveAttribute('href', expect.stringContaining('mlat=-6.200000&mlon=106.800000'))
  })
})
