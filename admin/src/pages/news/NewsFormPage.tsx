import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { Link, useNavigate, useParams } from 'react-router'
import { errorMessage } from '@/api/client'
import { newsApi, type NewsDetail } from '@/api/news'
import { ErrorSummary } from '@/components/cms/ErrorSummary'
import { ImageField } from '@/components/cms/ImageField'
import { CharCounter, SeoFields } from '@/components/cms/SeoFields'
import { SlugField } from '@/components/cms/SlugField'
import { UnsavedChangesDialog } from '@/components/cms/UnsavedChangesDialog'
import { RichTextEditor } from '@/components/editor/RichTextEditor'
import { Button } from '@/components/ui/Button'
import { SelectField, TextAreaField, TextField } from '@/components/ui/Field'
import { Card, PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { useUnsavedChanges } from '@/hooks/useUnsavedChanges'
import { localInputToDate } from '@/lib/datetime'
import { formatDateTime } from '@/lib/format'
import { willBeScheduled } from '@/lib/newsStatus'
import { previewSlug } from '@/lib/slug'
import { EXCERPT_MAX, NEWS_TIMEZONE, applyNewsErrors, emptyNews, newsSchema, newsToValues, valuesToInput, type NewsFormValues } from './newsForm'
import { NewsStatusBadge } from './NewsStatusBadge'

export const newsKeys = {
  detail: (id: string | undefined) => ['news', id] as const,
}

const STATUS_OPTIONS = [
  { value: 'draft', label: 'Draft' },
  { value: 'published', label: 'Published' },
]

export default function NewsFormPage() {
  const { id } = useParams()
  const news = useQuery({ queryKey: newsKeys.detail(id), queryFn: () => newsApi.get(id ?? ''), enabled: id !== undefined })
  if (id !== undefined && news.isPending) {
    return (
      <div className="flex justify-center p-12 text-navy-800">
        <Spinner size="lg" label="Loading article" />
      </div>
    )
  }
  if (news.isError) return <p role="alert" className="text-sm text-red-700">{errorMessage(news.error)}</p>
  return <NewsForm key={news.data?.updated_at ?? 'new'} news={news.data} />
}

export function NewsForm({ news }: { news: NewsDetail | undefined }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const [problems, setProblems] = useState<string[]>([])
  const [formError, setFormError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)

  const form = useForm<NewsFormValues>({
    resolver: zodResolver(newsSchema),
    defaultValues: news ? newsToValues(news) : emptyNews,
  })
  const { register, control, handleSubmit, setError, setValue, formState } = form
  const [title, slug, excerpt, status, publishLocal, seoTitle, metaDescription] = useWatch({
    control,
    name: ['title', 'slug', 'excerpt', 'status', 'publish_local', 'seo_title', 'meta_description'],
  })
  const { blocker, markLeaving } = useUnsavedChanges(formState.isDirty)

  const publishAt = localInputToDate(publishLocal, NEWS_TIMEZONE)
  const scheduled = willBeScheduled(status, publishAt)
  const futureDraft = status === 'draft' && publishAt !== null && publishAt.getTime() > Date.now()

  const invalidateLists = () => {
    void qc.invalidateQueries({ queryKey: ['grid', 'news'] })
  }

  const save = useMutation({
    mutationFn: (values: NewsFormValues) => {
      const input = valuesToInput(values)
      return news ? newsApi.update(news.id, input) : newsApi.create(input)
    },
    onSuccess: (saved) => {
      toast.success(news ? 'Article saved.' : 'Article created.')
      qc.setQueryData(newsKeys.detail(saved.id), saved)
      invalidateLists()
      if (!news) {
        markLeaving()
        void navigate(`/website/news/${saved.id}`, { replace: true })
      }
    },
    onError: (err) => {
      const result = applyNewsErrors(err, setError)
      setProblems(result.lines)
      setFormError(result.message)
    },
  })

  const remove = useMutation({
    mutationFn: () => newsApi.remove(news?.id ?? ''),
    onSuccess: () => {
      markLeaving()
      qc.removeQueries({ queryKey: newsKeys.detail(news?.id) })
      invalidateLists()
      toast.success('Article deleted.')
      void navigate('/website/news', { replace: true })
    },
    onError: (err) => {
      toast.error(errorMessage(err))
      setDeleting(false)
    },
  })

  const onSubmit = handleSubmit(
    (values) => {
      setProblems([])
      setFormError(null)
      save.mutate(values)
    },
    () => {
      setFormError('Fix the highlighted fields before saving.')
      setProblems([])
    },
  )

  const publicSlug = previewSlug(title, slug, news?.slug)
  const scheduleHint = scheduled
    ? `Scheduled: the article stays hidden until ${formatDateTime(publishAt?.toISOString(), NEWS_TIMEZONE)}.`
    : futureDraft
      ? 'Set the status to Published to schedule this article for that time.'
      : `Times are in ${NEWS_TIMEZONE}. Leave blank to publish as soon as the status is Published.`

  return (
    <>
      <PageHeader
        title={news ? news.title : 'New article'}
        description={news ? `Public address: /news/${news.slug}` : 'Write an article for the news page.'}
        actions={
          <>
            {news ? <NewsStatusBadge news={news} /> : null}
            <Link to="/website/news" className="text-sm text-navy-700 hover:underline">
              Back to news
            </Link>
          </>
        }
      />
      <form onSubmit={onSubmit} noValidate className="space-y-6">
        <ErrorSummary lines={problems} message={formError} />
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
          <div className="min-w-0 space-y-6">
            <Card title="Article">
              <div className="space-y-4">
                <TextField label="Title" required error={formState.errors.title?.message} {...register('title')} />
                <SlugField
                  title={title}
                  slug={slug}
                  existingSlug={news?.slug}
                  prefix="/news/"
                  error={formState.errors.slug?.message}
                  onAuto={(value) => setValue('slug', value, { shouldDirty: true })}
                  {...register('slug')}
                />
                <div>
                  <TextAreaField label="Excerpt" rows={3} error={formState.errors.excerpt?.message} hint="A short summary shown in the news list." {...register('excerpt')} />
                  <CharCounter value={excerpt} max={EXCERPT_MAX} label="Excerpt" />
                </div>
                <Controller
                  control={control}
                  name="body_html"
                  render={({ field, fieldState }) => (
                    <RichTextEditor label="Body" value={field.value} onChange={field.onChange} withLayout={false} minHeight="16rem" error={fieldState.error?.message} />
                  )}
                />
              </div>
            </Card>
          </div>
          <aside className="space-y-6">
            <Card title="Publishing">
              <div className="space-y-4">
                <SelectField label="Status" required options={STATUS_OPTIONS} error={formState.errors.status?.message} {...register('status')} />
                <TextField label="Publish date and time" type="datetime-local" error={formState.errors.publish_local?.message} hint={scheduleHint} {...register('publish_local')} />
              </div>
            </Card>
            <Card title="Thumbnail">
              <Controller
                control={control}
                name="thumbnail_file_id"
                render={({ field, fieldState }) => (
                  <ImageField
                    label="Thumbnail"
                    value={field.value}
                    onChange={field.onChange}
                    image={news?.thumbnail}
                    aspect="4 / 3"
                    hint="Also used as the social sharing image."
                    error={fieldState.error?.message}
                  />
                )}
              />
            </Card>
            <Card title="SEO">
              <SeoFields
                titleField={register('seo_title')}
                descriptionField={register('meta_description')}
                seoTitle={seoTitle}
                metaDescription={metaDescription}
                titleError={formState.errors.seo_title?.message}
                descriptionError={formState.errors.meta_description?.message}
                fallbackTitle={title}
                path={publicSlug === '' ? '/news/…' : `/news/${publicSlug}`}
              />
            </Card>
          </aside>
        </div>
        <div className="sticky bottom-0 z-10 -mx-1 flex flex-wrap items-center gap-2 border-t border-slate-200 bg-white/95 px-1 py-3">
          <Button type="submit" loading={save.isPending}>
            {news ? 'Save changes' : 'Create article'}
          </Button>
          {news ? (
            <Button variant="danger" className="ml-auto" onClick={() => setDeleting(true)}>
              Delete article
            </Button>
          ) : null}
        </div>
      </form>
      <ConfirmDialog
        open={deleting}
        title="Delete article?"
        message="The article disappears from the website."
        confirmLabel="Delete"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onClose={() => setDeleting(false)}
      />
      <UnsavedChangesDialog blocker={blocker} subject="article" />
    </>
  )
}
