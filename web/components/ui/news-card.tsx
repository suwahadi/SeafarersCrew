import Link from 'next/link'
import { Icon } from '@/components/ui/icons'
import { SiteImage } from '@/components/ui/site-image'
import { formatDate, isoDate } from '@/lib/format'
import type { NewsCard as NewsCardData } from '@/lib/types'

interface NewsCardProps {
  article: NewsCardData
  headingLevel?: 'h2' | 'h3'
}

export function NewsCard({ article, headingLevel: Heading = 'h3' }: NewsCardProps) {
  return (
    <article className="group relative flex flex-col overflow-hidden rounded-lg border border-line bg-white transition-shadow duration-200 ease-out hover:shadow-md">
      <div className="aspect-video overflow-hidden bg-surface">
        {article.thumbnail ? (
          <SiteImage
            image={article.thumbnail}
            sizes="(min-width: 1024px) 384px, (min-width: 640px) 50vw, 100vw"
            alt=""
            className="size-full object-cover transition-transform duration-300 ease-out group-hover:scale-105"
          />
        ) : (
          <div className="flex size-full items-center justify-center text-primary/30">
            <Icon name="anchor" className="size-12" />
          </div>
        )}
      </div>
      <div className="flex flex-1 flex-col p-5">
        <time dateTime={isoDate(article.published_at)} className="text-sm text-muted">
          {formatDate(article.published_at)}
        </time>
        <Heading className="mt-1 text-lg font-bold leading-snug text-primary">
          <Link href={`/news/${article.slug}`} className="after:absolute after:inset-0 hover:text-accent-text">
            {article.title}
          </Link>
        </Heading>
        {article.excerpt ? <p className="mt-2 line-clamp-3 text-sm text-muted">{article.excerpt}</p> : null}
      </div>
    </article>
  )
}
