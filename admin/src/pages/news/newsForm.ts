import type { UseFormSetError } from 'react-hook-form'
import { z } from 'zod'
import { ApiError, errorMessage } from '@/api/client'
import type { NewsDetail, NewsInput } from '@/api/news'
import { META_DESCRIPTION_MAX, SEO_TITLE_MAX } from '@/components/cms/SeoFields'
import { localInputToUtc, utcToLocalInput } from '@/lib/datetime'
import { DEFAULT_TIMEZONE } from '@/lib/format'
import { splitServerErrors, summarizeErrors } from '@/lib/serverErrors'

export const NEWS_TIMEZONE = DEFAULT_TIMEZONE
export const EXCERPT_MAX = 300

export const newsSchema = z.object({
  title: z.string().trim().min(1, 'The title is required.').max(200, 'The title must be at most 200 characters.'),
  slug: z.string().trim().max(120, 'The slug must be at most 120 characters.'),
  excerpt: z.string().max(EXCERPT_MAX, `The excerpt must be at most ${EXCERPT_MAX} characters.`),
  body_html: z.string(),
  thumbnail_file_id: z.string(),
  status: z.enum(['draft', 'published']),
  publish_local: z.string().refine((v) => v === '' || /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(v), 'Enter a valid date and time.'),
  seo_title: z.string().max(SEO_TITLE_MAX, `The SEO title must be at most ${SEO_TITLE_MAX} characters.`),
  meta_description: z.string().max(META_DESCRIPTION_MAX, `The meta description must be at most ${META_DESCRIPTION_MAX} characters.`),
})

export type NewsFormValues = z.infer<typeof newsSchema>

export const emptyNews: NewsFormValues = {
  title: '',
  slug: '',
  excerpt: '',
  body_html: '',
  thumbnail_file_id: '',
  status: 'draft',
  publish_local: '',
  seo_title: '',
  meta_description: '',
}

export const newsFormFields = ['title', 'slug', 'excerpt', 'body_html', 'thumbnail_file_id', 'status', 'publish_local', 'seo_title', 'meta_description'] as const

export function newsToValues(news: NewsDetail): NewsFormValues {
  return {
    title: news.title,
    slug: news.slug,
    excerpt: news.excerpt,
    body_html: news.body_html,
    thumbnail_file_id: news.thumbnail_file_id ?? '',
    status: news.status,
    publish_local: utcToLocalInput(news.published_at, NEWS_TIMEZONE),
    seo_title: news.seo_title,
    meta_description: news.meta_description,
  }
}

export function valuesToInput(values: NewsFormValues): NewsInput {
  return {
    title: values.title.trim(),
    slug: values.slug.trim(),
    excerpt: values.excerpt.trim(),
    body_html: values.body_html,
    thumbnail_file_id: values.thumbnail_file_id === '' ? null : values.thumbnail_file_id,
    published_at: values.publish_local === '' ? null : localInputToUtc(values.publish_local, NEWS_TIMEZONE),
    status: values.status,
    seo_title: values.seo_title.trim(),
    meta_description: values.meta_description.trim(),
  }
}

export function applyNewsErrors(err: unknown, setError: UseFormSetError<NewsFormValues>): { lines: string[]; message: string | null } {
  if (!(err instanceof ApiError)) return { lines: [], message: errorMessage(err) }
  const renamed = Object.fromEntries(Object.entries(err.fields).map(([key, message]) => [key === 'published_at' ? 'publish_local' : key, message]))
  const { matched, unmatched } = splitServerErrors(renamed, newsFormFields)
  for (const line of matched) setError(line.path as keyof NewsFormValues, { type: 'server', message: line.message })
  const lines = summarizeErrors([...matched, ...unmatched], [])
  return { lines, message: lines.length === 0 ? err.message : null }
}
