import type { Metadata } from 'next'
import { redirect } from 'next/navigation'
import { NewsCard } from '@/components/ui/news-card'
import { Container } from '@/components/ui/container'
import { PageHeader } from '@/components/ui/page-header'
import { Pagination } from '@/components/ui/pagination'
import { findPage, getNewsList, getSite } from '@/lib/api'
import { appUrl } from '@/lib/env'
import { pageHref, parsePageParam, totalPages } from '@/lib/pagination'
import { buildMetadata, pageTitle } from '@/lib/seo'
import { firstNonEmpty } from '@/lib/text'

const PER_PAGE = 9
const FALLBACK_TITLE = 'News'

interface NewsPageProps {
  searchParams: Promise<{ page?: string | string[] }>
}

export async function generateMetadata({ searchParams }: NewsPageProps): Promise<Metadata> {
  const current = parsePageParam((await searchParams).page)
  const [site, page] = await Promise.all([getSite(), findPage('news')])
  const base = firstNonEmpty(page?.seo_title, page?.title, FALLBACK_TITLE)
  return buildMetadata({
    site,
    baseUrl: appUrl(),
    path: pageHref('/news', current),
    title: pageTitle(site, current > 1 ? `${base} - Page ${String(current)}` : base),
    description: page?.meta_description,
    image: page?.og_image,
  })
}

export default async function NewsPage({ searchParams }: NewsPageProps) {
  const current = parsePageParam((await searchParams).page)
  const [list, page] = await Promise.all([getNewsList(current, PER_PAGE), findPage('news')])
  const pages = totalPages(list.meta.total, list.meta.per_page)
  if (current > pages) {
    redirect(pageHref('/news', pages))
  }

  const title = firstNonEmpty(page?.title, FALLBACK_TITLE)
  return (
    <>
      <PageHeader
        title={title}
        crumbs={[
          { name: 'Home', path: '/' },
          { name: title, path: '/news' },
        ]}
      />
      <Container className="py-12 sm:py-16">
        {list.data.length === 0 ? (
          <p className="rounded-lg border border-line bg-surface px-6 py-12 text-center text-muted">
            There are no news articles yet. Please check back soon.
          </p>
        ) : (
          <ul className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {list.data.map((article) => (
              <li key={article.slug}>
                <NewsCard article={article} headingLevel="h2" />
              </li>
            ))}
          </ul>
        )}
        <Pagination basePath="/news" current={current} pages={pages} label="News pages" />
      </Container>
    </>
  )
}
