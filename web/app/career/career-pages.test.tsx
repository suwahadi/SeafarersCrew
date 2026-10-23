import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { VacancyDetail } from '@/lib/api'
import type { Site, VacancyCard } from '@/lib/types'

const api = vi.hoisted(() => ({
  getSite: vi.fn(),
  findPage: vi.fn(),
  getVacancies: vi.fn(),
  getVacancyFilters: vi.fn(),
  findVacancy: vi.fn(),
  findRedirect: vi.fn(),
}))

vi.mock('@/lib/api', () => api)
vi.mock('next/link', () => ({
  default: ({ href, children, ...rest }: { href: string; children: ReactNode }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))
vi.mock('next/navigation', () => ({
  redirect: vi.fn((to: string) => {
    throw new Error(`redirect:${to}`)
  }),
  permanentRedirect: vi.fn((to: string) => {
    throw new Error(`permanent:${to}`)
  }),
  notFound: vi.fn(() => {
    throw new Error('not-found')
  }),
}))

const { default: CareerPage } = await import('@/app/career/page')
const { default: VacancyPage } = await import('@/app/career/[slug]/page')
const { default: ContactPage } = await import('@/app/contact/page')

const RANK = '01a15b9b-9bad-713d-97c1-ab02ab2dcc7b'

const SITE = {
  name: 'Seafarers Crew',
  turnstile_site_key: '',
  branding: { logo_light: null, logo_dark: null },
  organization: { legal_name: 'PT Seafarers', website: '', email: 'info@example.com', phone: '', address: 'Jakarta', country: 'ID' },
  contact: {
    office_address: 'Jl. Example 1',
    coordinates: { lat: -6.2, lng: 106.8 },
    phones: ['+62 21 0000'],
    email: 'info@example.com',
    whatsapp: { number: '1', message: '', url: 'https://wa.me/1' },
  },
  seo: { meta_title_pattern: '%s', meta_description: '', default_og_image: null },
} as unknown as Site

const CARD: VacancyCard = {
  id: 'ABCDEFGHI',
  title: 'Master AHTS',
  slug: 'master-ahts',
  rank: { id: RANK, name: 'Master' },
  vessel_type: { id: 'v', name: 'AHTS' },
  department: 'deck',
  location: 'Batam',
  employment_type: 'contract',
  closing_date: '2027-03-31',
  published_at: '2026-10-21T00:00:00Z',
}

const DETAIL: VacancyDetail = {
  ...CARD,
  duration: '4 months',
  salary_text: 'USD 9,000',
  description_html: '<p>About the job</p>',
  requirements_html: '<p>Needs COC</p>',
  required_documents: ['Passport'],
  is_open: true,
  updated_at: '2026-10-21T00:00:00Z',
}

const FILTERS = {
  ranks: [{ id: RANK, name: 'Master', short_code: 'MST' }],
  departments: ['deck'],
  vessel_types: [{ id: 'v', name: 'AHTS' }],
  locations: ['Batam'],
}

beforeEach(() => {
  Object.values(api).forEach((mock) => mock.mockReset())
  api.getSite.mockResolvedValue(SITE)
  api.findPage.mockResolvedValue(null)
  api.getVacancyFilters.mockResolvedValue(FILTERS)
})

describe('career page', () => {
  it('reflects url filters in the controls, sanitises bad params and lists results', async () => {
    api.getVacancies.mockResolvedValue({ data: [CARD], meta: { page: 1, per_page: 12, total: 1 } })
    render(
      await CareerPage({ searchParams: Promise.resolve({ q: 'master', rank: RANK, department: 'bogus', location: 'Batam' }) }),
    )
    expect(screen.getByLabelText('Keyword')).toHaveValue('master')
    expect(screen.getByLabelText('Rank')).toHaveValue(RANK)
    expect(screen.getByLabelText('Department')).toHaveValue('')
    expect(screen.getByLabelText('Location')).toHaveValue('Batam')
    expect(screen.getByRole('link', { name: 'Clear filters' })).toHaveAttribute('href', '/career')
    expect(screen.getByRole('status')).toHaveTextContent('1 vacancy found')
    expect(screen.getByRole('link', { name: 'Master AHTS' })).toHaveAttribute('href', '/career/master-ahts')
    expect(api.getVacancies).toHaveBeenCalledWith(expect.objectContaining({ q: 'master', rank: RANK, department: '', per_page: 12 }))
  })

  it('shows the empty state for a search without results', async () => {
    api.getVacancies.mockResolvedValue({ data: [], meta: { page: 1, per_page: 12, total: 0 } })
    render(await CareerPage({ searchParams: Promise.resolve({ q: 'zzz' }) }))
    expect(screen.getByText('No vacancies match your search.')).toBeInTheDocument()
  })

  it('keeps filters in pagination links', async () => {
    api.getVacancies.mockResolvedValue({ data: [CARD], meta: { page: 1, per_page: 12, total: 30 } })
    render(await CareerPage({ searchParams: Promise.resolve({ department: 'deck' }) }))
    expect(screen.getByRole('link', { name: 'Page 2' })).toHaveAttribute('href', '/career?department=deck&page=2')
  })
})

describe('vacancy page', () => {
  it('renders details and the required documents, and opens the application form from an Apply Now button', async () => {
    api.findVacancy.mockResolvedValue(DETAIL)
    const user = userEvent.setup()
    const { container } = render(await VacancyPage({ params: Promise.resolve({ slug: 'master-ahts' }) }))
    expect(screen.getByRole('heading', { level: 1, name: 'Master AHTS' })).toBeInTheDocument()
    expect(screen.getByText('Passport')).toBeInTheDocument()
    expect(screen.getByText('Needs COC')).toBeInTheDocument()
    const applyButtons = screen.getAllByRole('button', { name: 'Apply Now' })
    expect(applyButtons).toHaveLength(2)
    expect(screen.queryByRole('form', { name: 'Job application' })).not.toBeInTheDocument()
    await user.click(applyButtons[1]!)
    expect(screen.getByRole('form', { name: 'Job application' })).toBeInTheDocument()
    expect(container.innerHTML).toContain('"@type":"JobPosting"')
    expect(container.innerHTML).toContain('validThrough')
  })

  it('shows a closed notice and no form for a closed vacancy', async () => {
    api.findVacancy.mockResolvedValue({ ...DETAIL, is_open: false })
    const { container } = render(await VacancyPage({ params: Promise.resolve({ slug: 'master-ahts' }) }))
    expect(screen.getByText('This vacancy is closed')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Apply Now' })).not.toBeInTheDocument()
    expect(screen.queryByRole('form', { name: 'Job application' })).not.toBeInTheDocument()
    expect(container.innerHTML).not.toContain('JobPosting')
  })

  it('redirects an old slug and 404s an unknown one', async () => {
    api.findVacancy.mockResolvedValue(null)
    api.findRedirect.mockResolvedValueOnce({ from_path: '/career/old', to_path: '/career/new', status_code: 301 })
    await expect(VacancyPage({ params: Promise.resolve({ slug: 'old' }) })).rejects.toThrow('permanent:/career/new')
    expect(api.findRedirect).toHaveBeenCalledWith('/career/old')
    api.findRedirect.mockResolvedValueOnce(null)
    await expect(VacancyPage({ params: Promise.resolve({ slug: 'nope' }) })).rejects.toThrow('not-found')
  })
})

describe('contact page', () => {
  it('shows office details with tel, mailto and whatsapp links and the embedded map', async () => {
    render(await ContactPage())
    expect(screen.getByRole('link', { name: '+62 21 0000' })).toHaveAttribute('href', 'tel:+62210000')
    expect(screen.getByRole('link', { name: 'info@example.com' })).toHaveAttribute('href', 'mailto:info@example.com')
    expect(screen.getByRole('link', { name: 'Chat on WhatsApp' })).toHaveAttribute('href', 'https://wa.me/1')
    expect(document.querySelector('iframe')).not.toBeNull()
    expect(screen.getByRole('form', { name: 'Contact form' })).toBeInTheDocument()
  })

  it('omits the map block when coordinates are missing', async () => {
    api.getSite.mockResolvedValue({ ...SITE, contact: { ...SITE.contact, coordinates: null } })
    render(await ContactPage())
    expect(document.querySelector('iframe')).toBeNull()
  })
})
