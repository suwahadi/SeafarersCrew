import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { BlockRenderer } from '@/components/blocks/block-renderer'
import { HomeSections } from '@/components/blocks/home/home-sections'
import { CountUp } from '@/components/ui/count-up'
import { FaqAccordion } from '@/components/ui/faq-accordion'
import type { Home, RawBlock } from '@/lib/types'

vi.mock('next/link', () => ({
  default: ({ href, children, ...rest }: { href: string; children: ReactNode }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))

describe('BlockRenderer', () => {
  const blocks: RawBlock[] = [
    { id: '1', type: 'rich_text', settings: { html: '<p>Hello rich</p>' } },
    { id: '2', type: 'faq', settings: { items: [{ question: 'Why?', answer_html: '<p>Because</p>' }] } },
    { id: '3', type: 'stats', settings: { items: [{ number: 12, suffix: '+', label: 'Ships' }] } },
    { id: '4', type: 'cta', settings: { heading: 'Join', text: '', button_label: 'Go', button_link: '/career', background_image: null, background_color: '' } },
    { id: '5', type: 'card_grid', settings: { columns: 3, cards: [{ icon: 'ship', image: null, title: 'Card', text: '', link: '' }] } },
    { id: '6', type: 'video', settings: { provider: 'youtube', video_id: 'abcdefghijk', title: 'Film', url: '', embed_url: '', thumbnail_url: '' } },
    { id: '7', type: 'image_text', settings: { image: null, heading: 'Split', text_html: '<p>x</p>', button_label: '', button_link: '', image_position: 'left' } },
    { id: '8', type: 'gallery', settings: { images: null, columns: 3, lightbox: true } },
    { id: '9', type: 'mystery', settings: {} },
    { id: '10', type: 'rich_text', settings: null },
  ]

  it('renders the matching component per type, skips empty galleries and ignores unknown or malformed blocks', () => {
    const { container } = render(<BlockRenderer blocks={blocks} />)
    const rendered = [...container.querySelectorAll('[data-block]')].map((node) => node.getAttribute('data-block'))
    expect(rendered).toEqual(['rich_text', 'faq', 'stats', 'cta', 'card_grid', 'video', 'image_text', 'gallery'])
    expect(screen.getByText('Hello rich')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Why?' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Join' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Play video/ })).toBeInTheDocument()
  })

  it('renders nothing for null blocks', () => {
    const { container } = render(<BlockRenderer blocks={null} />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('FaqAccordion', () => {
  it('toggles aria-expanded and makes the answer inert when closed', async () => {
    render(<FaqAccordion items={[{ question: 'Q1', answerHtml: '<p>A1</p>' }]} />)
    const button = screen.getByRole('button', { name: 'Q1' })
    expect(button).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByText('A1').closest('[inert]')).not.toBeNull()
    await userEvent.click(button)
    expect(button).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('A1').closest('[inert]')).toBeNull()
    await userEvent.click(button)
    expect(button).toHaveAttribute('aria-expanded', 'false')
  })
})

describe('CountUp', () => {
  const originalMatchMedia = window.matchMedia

  afterEach(() => {
    window.matchMedia = originalMatchMedia
    vi.unstubAllGlobals()
  })

  it('renders the final value on the server-rendered markup', () => {
    render(<CountUp value={1250} suffix="+" />)
    expect(screen.getByText('1,250')).toBeInTheDocument()
    expect(screen.getByText('1,250').parentElement).toHaveTextContent('1,250+')
  })

  it('keeps the final value under reduced motion', () => {
    window.matchMedia = vi.fn().mockReturnValue({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() })
    render(<CountUp value={500} />)
    expect(screen.getByText('500')).toBeInTheDocument()
  })

  describe('with an intersection observer', () => {
    let trigger: () => void = () => undefined

    beforeEach(() => {
      vi.useFakeTimers()
      window.matchMedia = vi.fn().mockReturnValue({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() })
      class FakeObserver {
        constructor(callback: (entries: { isIntersecting: boolean }[]) => void) {
          trigger = () => {
            callback([{ isIntersecting: true }])
          }
        }
        observe = vi.fn()
        disconnect = vi.fn()
      }
      vi.stubGlobal('IntersectionObserver', FakeObserver)
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('starts at zero then ends on the final value once visible', () => {
      render(<CountUp value={80} />)
      expect(screen.getByText('0')).toBeInTheDocument()
      act(() => {
        trigger()
        vi.advanceTimersByTime(2000)
      })
      expect(screen.getByText('80')).toBeInTheDocument()
    })
  })
})

describe('HomeSections', () => {
  const emptyHome: Home = {
    page: { title: 'Home', seo_title: '', meta_description: '', og_image: null, intro_html: '<p>Intro text</p>' },
    sections: ['cta', 'intro', 'news', 'stats', 'bogus', 'hero', 'gallery', 'testimonials', 'clients', 'vacancies', 'services'],
    hero: { autoplay_seconds: 5, slides: [] },
    stats: [{ id: 's', number: 5, suffix: '', label: 'Ships' }],
    news: [],
    vacancies: null,
    gallery: [],
    testimonials: [],
    client_logos: [],
  }

  it('follows the configured order, skips empty sections and unknown keys', () => {
    const { container } = render(<HomeSections home={emptyHome} siteName="Seafarers Crew" services={null} />)
    const keys = [...container.querySelectorAll('[data-section]')].map((node) => node.getAttribute('data-section'))
    expect(keys).toEqual(['cta', 'intro', 'stats'])
  })

  it('renders services when a card grid is available and adds a hidden h1 without a hero', () => {
    render(
      <HomeSections
        home={{ ...emptyHome, sections: ['services'] }}
        siteName="Seafarers Crew"
        services={{ title: 'Our Services', settings: { columns: 3, cards: [{ icon: 'ship', image: null, title: 'Crew supply', text: '', link: '' }] } }}
      />,
    )
    expect(screen.getByRole('heading', { level: 1, name: 'Seafarers Crew' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Crew supply' })).toBeInTheDocument()
  })

  it('renders the hero with a single h1 and eager high-priority first image only', () => {
    const img = (url: string) => ({ url, mime_type: 'image/webp', width: 1600, height: 700, alt: 'a', sources: [] })
    const slide = (id: string, image: string) => ({ id, heading: `Slide ${id}`, subheading: '', button_label: '', button_link: '', image: img(image), mobile_image: null })
    const { container } = render(
      <HomeSections
        home={{ ...emptyHome, sections: ['hero'], hero: { autoplay_seconds: 5, slides: [slide('1', 'one.webp'), slide('2', 'two.webp')] } }}
        siteName="Seafarers Crew"
        services={null}
      />,
    )
    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1)
    const images = [...container.querySelectorAll('img')]
    expect(images[0]).toHaveAttribute('fetchpriority', 'high')
    expect(images[0]).toHaveAttribute('loading', 'eager')
    expect(images[1]).toHaveAttribute('loading', 'lazy')
    expect(images[1]).not.toHaveAttribute('fetchpriority')
  })
})
