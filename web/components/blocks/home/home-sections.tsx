import type { ReactNode } from 'react'
import { ClientsSection } from '@/components/blocks/home/clients-section'
import { CtaSection } from '@/components/blocks/home/cta-section'
import { GallerySection } from '@/components/blocks/home/gallery-section'
import { HeroSection } from '@/components/blocks/home/hero-section'
import { IntroSection } from '@/components/blocks/home/intro-section'
import { NewsSection } from '@/components/blocks/home/news-section'
import { ServicesSection } from '@/components/blocks/home/services-section'
import { StatsSection } from '@/components/blocks/home/stats-section'
import { TestimonialsSection } from '@/components/blocks/home/testimonials-section'
import { VacanciesSection } from '@/components/blocks/home/vacancies-section'
import type { ServicesGrid } from '@/lib/api'
import { visibleSections } from '@/lib/home'
import { list } from '@/lib/list'
import type { Home, HomeSectionKey } from '@/lib/types'

interface HomeSectionsProps {
  home: Home
  siteName: string
  services: ServicesGrid | null
}

export function HomeSections({ home, siteName, services }: HomeSectionsProps) {
  const sections: Record<HomeSectionKey, () => ReactNode> = {
    hero: () => <HeroSection slides={list(home.hero.slides)} autoplaySeconds={home.hero.autoplay_seconds} />,
    intro: () => <IntroSection html={home.page?.intro_html ?? ''} />,
    stats: () => <StatsSection items={list(home.stats)} />,
    services: () => (services ? <ServicesSection services={services} /> : null),
    news: () => <NewsSection articles={list(home.news)} />,
    vacancies: () => <VacanciesSection vacancies={list(home.vacancies)} />,
    gallery: () => <GallerySection items={list(home.gallery)} />,
    testimonials: () => <TestimonialsSection testimonials={list(home.testimonials)} />,
    clients: () => <ClientsSection clients={list(home.client_logos)} />,
    cta: () => <CtaSection siteName={siteName} />,
  }
  const keys = visibleSections(home, services !== null)

  return (
    <>
      {keys.includes('hero') ? null : <h1 className="sr-only">{siteName}</h1>}
      {keys.map((key) => (
        <div key={key} data-section={key}>
          {sections[key]()}
        </div>
      ))}
    </>
  )
}
