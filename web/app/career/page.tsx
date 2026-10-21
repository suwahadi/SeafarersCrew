import type { Metadata } from 'next'
import { redirect } from 'next/navigation'
import { CareerFilters } from '@/components/career/career-filters'
import { Container } from '@/components/ui/container'
import { PageHeader } from '@/components/ui/page-header'
import { Pagination } from '@/components/ui/pagination'
import { VacancyCard } from '@/components/ui/vacancy-card'
import { findPage, getSite, getVacancies, getVacancyFilters } from '@/lib/api'
import { careerHref, hasActiveFilters, parseCareerQuery, toVacancyQuery } from '@/lib/career'
import { appUrl } from '@/lib/env'
import { totalPages } from '@/lib/pagination'
import { buildMetadata, pageTitle } from '@/lib/seo'
import { firstNonEmpty } from '@/lib/text'

const FALLBACK_TITLE = 'Career'

interface CareerPageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}

export async function generateMetadata({ searchParams }: CareerPageProps): Promise<Metadata> {
  const query = parseCareerQuery(await searchParams)
  const filtered = hasActiveFilters(query)
  const [site, page] = await Promise.all([getSite(), findPage('career')])
  const base = firstNonEmpty(page?.seo_title, page?.title, FALLBACK_TITLE)
  const metadata = buildMetadata({
    site,
    baseUrl: appUrl(),
    path: filtered ? '/career' : careerHref(query, query.page),
    title: pageTitle(site, query.page > 1 ? `${base} - Page ${String(query.page)}` : base),
    description: page?.meta_description,
    image: page?.og_image,
  })
  return filtered ? { ...metadata, robots: { index: false, follow: true } } : metadata
}

export default async function CareerPage({ searchParams }: CareerPageProps) {
  const query = parseCareerQuery(await searchParams)
  const [list, options, page] = await Promise.all([
    getVacancies(toVacancyQuery(query)),
    getVacancyFilters(),
    findPage('career'),
  ])
  const pages = totalPages(list.meta.total, list.meta.per_page)
  if (query.page > pages) {
    redirect(careerHref(query, pages))
  }

  const title = firstNonEmpty(page?.title, FALLBACK_TITLE)
  const filtered = hasActiveFilters(query)
  const total = list.meta.total
  return (
    <>
      <PageHeader
        title={title}
        introHtml={page?.intro_html}
        crumbs={[
          { name: 'Home', path: '/' },
          { name: title, path: '/career' },
        ]}
      />
      <Container className="py-10 sm:py-14">
        <CareerFilters filters={query} options={options} />
        <p role="status" className="mb-6 mt-8 text-sm font-medium text-muted">
          {total === 0 ? 'No vacancies found' : `${String(total)} ${total === 1 ? 'vacancy' : 'vacancies'} found`}
        </p>
        {list.data.length === 0 ? (
          <div className="rounded-lg border border-line bg-surface px-6 py-12 text-center">
            <p className="text-lg font-semibold text-primary">
              {filtered ? 'No vacancies match your search.' : 'There are no open vacancies at the moment.'}
            </p>
            <p className="mt-2 text-muted">
              {filtered
                ? 'Try different keywords or clear the filters to see all vacancies.'
                : 'Please check back soon or get in touch with our crewing team.'}
            </p>
          </div>
        ) : (
          <ul className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {list.data.map((vacancy) => (
              <li key={vacancy.id}>
                <VacancyCard vacancy={vacancy} headingLevel="h2" />
              </li>
            ))}
          </ul>
        )}
        <Pagination
          basePath="/career"
          current={query.page}
          pages={pages}
          label="Vacancy pages"
          hrefFor={(target) => careerHref(query, target)}
        />
      </Container>
    </>
  )
}
