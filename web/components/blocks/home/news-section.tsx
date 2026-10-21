import Link from 'next/link'
import { Container } from '@/components/ui/container'
import { NewsCard } from '@/components/ui/news-card'
import { SectionHeading } from '@/components/ui/section-heading'
import type { NewsCard as NewsCardData } from '@/lib/types'

export function NewsSection({ articles }: { articles: readonly NewsCardData[] }) {
  return (
    <section aria-labelledby="home-news" className="py-16 sm:py-20">
      <Container>
        <SectionHeading
          id="home-news"
          eyebrow="News"
          title="Latest news"
          action={
            <Link href="/news" className="text-sm font-semibold text-accent-text hover:underline">
              All news
            </Link>
          }
        />
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {articles.map((article) => (
            <NewsCard key={article.slug} article={article} />
          ))}
        </div>
      </Container>
    </section>
  )
}
