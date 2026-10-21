import type { Metadata } from 'next'
import { Container } from '@/components/ui/container'
import { JsonLd } from '@/components/ui/json-ld'
import { NewsCard } from '@/components/ui/news-card'
import { PageHeader } from '@/components/ui/page-header'
import { RichHtml } from '@/components/ui/rich-html'
import { ShareButtons } from '@/components/ui/share-buttons'
import { SiteImage } from '@/components/ui/site-image'
import { findArticle, getSite } from '@/lib/api'
import { appUrl } from '@/lib/env'
import { formatDate, isoDate } from '@/lib/format'
import { newsArticleJsonLd } from '@/lib/jsonld'
import { list } from '@/lib/list'
import { notFoundOrRedirect } from '@/lib/redirects'
import { buildMetadata, canonicalUrl, pageTitle } from '@/lib/seo'
import { firstNonEmpty } from '@/lib/text'

interface ArticlePageProps {
  params: Promise<{ slug: string }>
}

export async function generateMetadata({ params }: ArticlePageProps): Promise<Metadata> {
  const { slug } = await params
  const [site, article] = await Promise.all([getSite(), findArticle(slug)])
  if (!article) {
    return { title: 'Article not found', robots: { index: false } }
  }
  return buildMetadata({
    site,
    baseUrl: appUrl(),
    path: `/news/${article.slug}`,
    title: pageTitle(site, firstNonEmpty(article.seo_title, article.title)),
    description: firstNonEmpty(article.meta_description, article.excerpt),
    image: article.thumbnail,
    type: 'article',
    publishedTime: article.published_at,
    modifiedTime: article.updated_at,
  })
}

export default async function ArticlePage({ params }: ArticlePageProps) {
  const { slug } = await params
  const [site, article] = await Promise.all([getSite(), findArticle(slug)])
  if (!article) {
    return notFoundOrRedirect(`news/${slug}`)
  }
  const related = list(article.related)
  const url = canonicalUrl(appUrl(), `/news/${article.slug}`)

  return (
    <>
      <PageHeader
        title={article.title}
        crumbs={[
          { name: 'Home', path: '/' },
          { name: 'News', path: '/news' },
          { name: article.title, path: `/news/${article.slug}` },
        ]}
      >
        <p className="mt-4 text-sm opacity-85">
          <time dateTime={isoDate(article.published_at)}>{formatDate(article.published_at)}</time>
        </p>
      </PageHeader>
      <Container className="py-12 sm:py-16">
        <article className="mx-auto max-w-3xl">
          {article.thumbnail ? (
            <SiteImage
              image={article.thumbnail}
              sizes="(min-width: 768px) 768px, 100vw"
              priority
              alt={firstNonEmpty(article.thumbnail.alt, article.title)}
              className="mb-8 w-full rounded-lg object-cover"
            />
          ) : null}
          <RichHtml html={article.body_html} className="prose-wide" />
          <div className="mt-10 border-t border-line pt-6">
            <ShareButtons url={url} title={article.title} />
          </div>
        </article>
      </Container>
      {related.length > 0 ? (
        <section aria-labelledby="related-news" className="bg-surface py-12 sm:py-16">
          <Container>
            <h2 id="related-news" className="mb-8 text-2xl font-bold tracking-tight text-primary">
              Related news
            </h2>
            <ul className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {related.map((item) => (
                <li key={item.slug}>
                  <NewsCard article={item} />
                </li>
              ))}
            </ul>
          </Container>
        </section>
      ) : null}
      <JsonLd node={newsArticleJsonLd(article, site, appUrl())} />
    </>
  )
}
