import type { Metadata } from 'next'
import Link from 'next/link'
import { ApplyButton, ApplyProvider } from '@/components/forms/ApplyDialog'
import { Container } from '@/components/ui/container'
import { JsonLd } from '@/components/ui/json-ld'
import { PageHeader } from '@/components/ui/page-header'
import { RichHtml } from '@/components/ui/rich-html'
import { findVacancy, getSite, getVacancyFilters, type VacancyDetail } from '@/lib/api'
import { appUrl, publicApiUrl } from '@/lib/env'
import { departmentLabel, employmentLabel, formatDate } from '@/lib/format'
import { jobPostingJsonLd } from '@/lib/jsonld'
import { list } from '@/lib/list'
import { notFoundOrRedirect } from '@/lib/redirects'
import { buildMetadata, pageTitle } from '@/lib/seo'
import { excerptFromHtml } from '@/lib/text'

interface VacancyPageProps {
  params: Promise<{ slug: string }>
}

function rankOptions(vacancy: VacancyDetail, ranks: readonly { id: string; name: string }[]) {
  const options = ranks.map((rank) => ({ id: rank.id, name: rank.name }))
  if (!options.some((rank) => rank.id === vacancy.rank.id)) {
    options.push({ id: vacancy.rank.id, name: vacancy.rank.name })
  }
  return options.sort((a, b) => a.name.localeCompare(b.name))
}

function summaryRows(vacancy: VacancyDetail): { label: string; value: string }[] {
  const rows = [
    { label: 'Vacancy ID', value: vacancy.id },
    { label: 'Rank', value: vacancy.rank.name },
    { label: 'Vessel type', value: vacancy.vessel_type?.name ?? '' },
    { label: 'Department', value: departmentLabel(vacancy.department) },
    { label: 'Location', value: vacancy.location },
    { label: 'Employment type', value: employmentLabel(vacancy.employment_type) },
    { label: 'Duration', value: vacancy.duration },
    { label: 'Salary', value: vacancy.salary_text },
    { label: 'Closing date', value: vacancy.closing_date ? formatDate(vacancy.closing_date) : 'Open until filled' },
  ]
  return rows.filter((row) => row.value !== '')
}

function closingText(vacancy: VacancyDetail): string {
  return vacancy.closing_date ? `Applications close on ${formatDate(vacancy.closing_date)}` : 'Open until filled'
}

export async function generateMetadata({ params }: VacancyPageProps): Promise<Metadata> {
  const { slug } = await params
  const [site, vacancy] = await Promise.all([getSite(), findVacancy(slug)])
  if (!vacancy) {
    return { title: 'Vacancy not found', robots: { index: false } }
  }
  const metadata = buildMetadata({
    site,
    baseUrl: appUrl(),
    path: `/career/${vacancy.slug}`,
    title: pageTitle(site, `${vacancy.title} (${vacancy.id})`),
    description: excerptFromHtml(vacancy.description_html) || `${vacancy.title} - ${vacancy.rank.name}`,
  })
  return vacancy.is_open ? metadata : { ...metadata, robots: { index: false, follow: true } }
}

export default async function VacancyPage({ params }: VacancyPageProps) {
  const { slug } = await params
  const [site, vacancy, filters] = await Promise.all([getSite(), findVacancy(slug), getVacancyFilters()])
  if (!vacancy) {
    return notFoundOrRedirect(`/career/${slug}`)
  }
  const documents = list(vacancy.required_documents)

  const page = (
    <>
      <PageHeader
        title={vacancy.title}
        crumbs={[
          { name: 'Home', path: '/' },
          { name: 'Career', path: '/career' },
          { name: vacancy.title, path: `/career/${vacancy.slug}` },
        ]}
      >
        <div className="mt-6 flex flex-wrap items-center gap-x-5 gap-y-3">
          {vacancy.is_open ? (
            <>
              <ApplyButton />
              <p className="text-sm opacity-90">{closingText(vacancy)}</p>
            </>
          ) : (
            <p className="rounded-full bg-white/15 px-4 py-1.5 text-sm font-semibold">Closed</p>
          )}
        </div>
      </PageHeader>
      <Container className="grid gap-10 py-10 sm:py-14 lg:grid-cols-[1fr_20rem] lg:gap-12">
        <div className="min-w-0 space-y-10">
          {vacancy.is_open ? null : (
            <div role="status" className="rounded-lg border border-amber-400 bg-amber-50 px-5 py-4 text-amber-950">
              <p className="font-bold">This vacancy is closed</p>
              <p className="mt-1 text-sm">
                We are no longer accepting applications for this position.{' '}
                <Link href="/career" className="font-semibold underline underline-offset-2">
                  View current vacancies
                </Link>
              </p>
            </div>
          )}
          {vacancy.description_html.trim() !== '' ? (
            <section aria-labelledby="vacancy-description">
              <h2 id="vacancy-description" className="mb-4 text-2xl font-bold text-primary">
                Job description
              </h2>
              <RichHtml html={vacancy.description_html} />
            </section>
          ) : null}
          {vacancy.requirements_html.trim() !== '' ? (
            <section aria-labelledby="vacancy-requirements">
              <h2 id="vacancy-requirements" className="mb-4 text-2xl font-bold text-primary">
                Requirements
              </h2>
              <RichHtml html={vacancy.requirements_html} />
            </section>
          ) : null}
          {documents.length > 0 ? (
            <section aria-labelledby="vacancy-documents">
              <h2 id="vacancy-documents" className="mb-4 text-2xl font-bold text-primary">
                Required documents
              </h2>
              <ul className="list-disc space-y-1.5 pl-6 marker:text-accent-text">
                {documents.map((document) => (
                  <li key={document}>{document}</li>
                ))}
              </ul>
            </section>
          ) : null}
          {vacancy.is_open ? (
            <section aria-labelledby="vacancy-apply" className="rounded-lg border border-line bg-surface p-6 sm:p-8">
              <h2 id="vacancy-apply" className="text-xl font-bold text-primary">
                Apply for this position
              </h2>
              <p className="mt-2 text-muted">Have your CV ready. Certificates and other supporting documents are optional.</p>
              <div className="mt-5 flex flex-wrap items-center gap-x-5 gap-y-3">
                <ApplyButton />
                <p className="text-sm text-muted">{closingText(vacancy)}</p>
              </div>
            </section>
          ) : null}
        </div>
        <aside aria-labelledby="vacancy-summary" className="h-fit rounded-lg border border-line bg-white p-5 lg:sticky lg:top-24">
          <h2 id="vacancy-summary" className="mb-4 text-lg font-bold text-primary">
            Job summary
          </h2>
          <dl className="divide-y divide-line text-sm">
            {summaryRows(vacancy).map((row) => (
              <div key={row.label} className="py-2.5 first:pt-0 last:pb-0">
                <dt className="text-muted">{row.label}</dt>
                <dd translate={row.label === 'Vacancy ID' ? 'no' : undefined} className="mt-0.5 font-semibold text-ink">
                  {row.value}
                </dd>
              </div>
            ))}
          </dl>
        </aside>
      </Container>
    </>
  )

  if (!vacancy.is_open) {
    return page
  }

  return (
    <>
      <ApplyProvider
        vacancyTitle={vacancy.title}
        vacancyId={vacancy.id}
        form={{
          vacancySlug: vacancy.slug,
          ranks: rankOptions(vacancy, filters.ranks),
          defaultRankId: vacancy.rank.id,
          turnstileSiteKey: site.turnstile_site_key,
          apiUrl: publicApiUrl(),
        }}
      >
        {page}
      </ApplyProvider>
      <JsonLd node={jobPostingJsonLd(vacancy, site, appUrl())} />
    </>
  )
}
