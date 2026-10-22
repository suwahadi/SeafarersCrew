import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Controller, FormProvider, useForm, useWatch, type FieldPath, type UseFormSetError } from 'react-hook-form'
import { Link, useNavigate, useParams } from 'react-router'
import { ApiError, errorMessage } from '@/api/client'
import { pagesApi, type PageDetail, type PageMedia } from '@/api/pages'
import { ErrorSummary } from '@/components/cms/ErrorSummary'
import { ImageField } from '@/components/cms/ImageField'
import { SeoFields } from '@/components/cms/SeoFields'
import { SlugField } from '@/components/cms/SlugField'
import { UnsavedChangesDialog } from '@/components/cms/UnsavedChangesDialog'
import { RichTextEditor } from '@/components/editor/RichTextEditor'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { CheckboxField, SelectField, TextField } from '@/components/ui/Field'
import { Badge, Card, PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { useUnsavedChanges } from '@/hooks/useUnsavedChanges'
import type { FormBlock } from '@/lib/pageBlocks'
import { emptyPageValues, pageToValues, valuesToPayload, type PageFormValues } from '@/lib/pagePayload'
import { collectErrorLines, splitServerErrors, summarizeErrors } from '@/lib/serverErrors'
import { previewSlug } from '@/lib/slug'
import { BlockBuilder } from './BlockBuilder'
import { pageSchema, pageTopFields } from './pageForm'

export const pageKeys = {
  detail: (id: string | undefined) => ['page', id] as const,
}

const STATUS_OPTIONS = [
  { value: 'draft', label: 'Draft' },
  { value: 'published', label: 'Published' },
]

export function applyPageErrors(err: unknown, setError: UseFormSetError<PageFormValues>, blocks: readonly Pick<FormBlock, 'type'>[]): { lines: string[]; message: string | null } {
  if (!(err instanceof ApiError)) return { lines: [], message: errorMessage(err) }
  const { matched, unmatched } = splitServerErrors(err.fields, pageTopFields)
  for (const line of matched) setError(line.path as FieldPath<PageFormValues>, { type: 'server', message: line.message })
  const lines = summarizeErrors([...matched, ...unmatched], blocks)
  return { lines, message: lines.length === 0 ? err.message : null }
}

export default function PageFormPage() {
  const { id } = useParams()
  const page = useQuery({ queryKey: pageKeys.detail(id), queryFn: () => pagesApi.get(id ?? ''), enabled: id !== undefined })
  if (id !== undefined && page.isPending) {
    return (
      <div className="flex justify-center p-12 text-navy-800">
        <Spinner size="lg" label="Loading page" />
      </div>
    )
  }
  if (page.isError) return <p role="alert" className="text-sm text-red-700">{errorMessage(page.error)}</p>
  return <PageForm key={page.data?.updated_at ?? 'new'} page={page.data} />
}

export function PageForm({ page }: { page: PageDetail | undefined }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const systemPage = page?.is_system === true ? page : undefined
  const isSystem = systemPage !== undefined
  const [problems, setProblems] = useState<string[]>([])
  const [formError, setFormError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)

  const form = useForm<PageFormValues>({
    resolver: zodResolver(pageSchema),
    defaultValues: page ? pageToValues(page) : emptyPageValues,
  })
  const { register, control, handleSubmit, setError, setValue, getValues, formState } = form
  const [title, slug, seoTitle, metaDescription] = useWatch({ control, name: ['title', 'slug', 'seo_title', 'meta_description'] })
  const dirty = formState.isDirty
  const { blocker, markLeaving } = useUnsavedChanges(dirty)

  const media: PageMedia = { ...page?.media }
  if (page?.og_image) media[page.og_image.file_id] = page.og_image

  const invalidateLists = () => {
    void qc.invalidateQueries({ queryKey: ['grid', 'pages'] })
  }

  const save = useMutation({
    mutationFn: (values: PageFormValues) => {
      const input = valuesToPayload(values)
      return page ? pagesApi.update(page.id, input) : pagesApi.create(input)
    },
    onSuccess: (saved) => {
      toast.success(page ? 'Page saved.' : 'Page created.')
      qc.setQueryData(pageKeys.detail(saved.id), saved)
      invalidateLists()
      if (!page) {
        markLeaving()
        void navigate(`/website/pages/${saved.id}`, { replace: true })
      }
    },
    onError: (err) => {
      const result = applyPageErrors(err, setError, getValues('blocks'))
      setProblems(result.lines)
      setFormError(result.message)
    },
  })

  const remove = useMutation({
    mutationFn: () => pagesApi.remove(page?.id ?? ''),
    onSuccess: () => {
      markLeaving()
      qc.removeQueries({ queryKey: pageKeys.detail(page?.id) })
      invalidateLists()
      toast.success('Page deleted.')
      void navigate('/website/pages', { replace: true })
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
    (errors) => {
      setFormError(null)
      setProblems(summarizeErrors(collectErrorLines(errors), getValues('blocks')))
    },
  )

  const publicSlug = previewSlug(title, slug, page?.slug)
  const path = page ? page.path : publicSlug === '' ? '/…' : `/${publicSlug}`

  return (
    <>
      <PageHeader
        title={page ? page.title : 'New page'}
        description={page ? `Public address: ${page.path}` : 'Build a page for the website.'}
        actions={
          <>
            {page ? <Badge tone={page.status === 'published' ? 'green' : 'neutral'}>{page.status === 'published' ? 'Published' : 'Draft'}</Badge> : null}
            {isSystem ? <Badge tone="blue">System page</Badge> : null}
            <Link to="/website/pages" className="text-sm text-navy-700 hover:underline">
              Back to pages
            </Link>
          </>
        }
      />
      <FormProvider {...form}>
        <form id="page-form" onSubmit={onSubmit} noValidate className="space-y-6">
          <ErrorSummary lines={problems} message={formError} />
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
            <div className="min-w-0 space-y-6">
              <Card title="Intro text">
                <Controller
                  control={control}
                  name="intro_html"
                  render={({ field, fieldState }) => (
                    <RichTextEditor label="Intro text" value={field.value} onChange={field.onChange} withLayout={false} minHeight="8rem" error={fieldState.error?.message} />
                  )}
                />
              </Card>
              {isSystem ? (
                <div role="note" className="rounded-lg bg-blue-50 p-4 text-sm text-blue-900 ring-1 ring-blue-200">
                  <p className="font-medium">This is a system page.</p>
                  <p className="mt-1">Its layout is fixed. You can edit the title, intro text, SEO fields and menu settings, but not the slug, status or blocks.</p>
                </div>
              ) : (
                <Card title="Page blocks">
                  <BlockBuilder media={media} />
                </Card>
              )}
            </div>
            <aside className="space-y-6">
              <Card title="Page settings">
                <div className="space-y-4">
                  <TextField label="Title" required error={formState.errors.title?.message} {...register('title')} />
                  {systemPage ? (
                    <>
                      <TextField label="Slug" value={systemPage.slug} disabled readOnly hint="System page addresses are fixed." onChange={() => undefined} />
                      <TextField label="Status" value={systemPage.status === 'published' ? 'Published' : 'Draft'} disabled readOnly onChange={() => undefined} />
                    </>
                  ) : (
                    <>
                      <SlugField
                        title={title}
                        slug={slug}
                        existingSlug={page?.slug}
                        prefix="/"
                        error={formState.errors.slug?.message}
                        onAuto={(value) => setValue('slug', value, { shouldDirty: true })}
                        {...register('slug')}
                      />
                      <SelectField label="Status" required options={STATUS_OPTIONS} error={formState.errors.status?.message} {...register('status')} />
                    </>
                  )}
                </div>
              </Card>
              <Card title="Menus">
                <div className="space-y-3">
                  <CheckboxField label="Show in header menu" {...register('in_header_menu')} />
                  <CheckboxField label="Show in footer menu" {...register('in_footer_menu')} />
                  <TextField
                    label="Menu order"
                    type="number"
                    min={0}
                    max={10000}
                    hint="Lower numbers come first."
                    error={formState.errors.menu_order?.message}
                    {...register('menu_order', { valueAsNumber: true })}
                  />
                </div>
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
                  path={path}
                />
              </Card>
              <Card title="Social image">
                <Controller
                  control={control}
                  name="og_image_file_id"
                  render={({ field, fieldState }) => (
                    <ImageField
                      label="Open Graph image"
                      value={field.value}
                      onChange={field.onChange}
                      image={media[field.value]}
                      hint="Shown when the page is shared. Optional."
                      error={fieldState.error?.message}
                    />
                  )}
                />
              </Card>
            </aside>
          </div>
          <div className="sticky bottom-0 z-10 -mx-1 flex items-center gap-2 max-sm:*:first:flex-1 border-t border-slate-200 bg-white/95 px-1 py-3">
            <Button type="submit" loading={save.isPending}>
              {page ? 'Save changes' : 'Create page'}
            </Button>
            {page && !isSystem ? (
              <Button variant="danger" className="ml-auto" collapse icon={<Icon name="trash" />} onClick={() => setDeleting(true)}>
                Delete page
              </Button>
            ) : null}
          </div>
        </form>
      </FormProvider>
      <ConfirmDialog
        open={deleting}
        title="Delete page?"
        message="The page disappears from the website and from its menus."
        confirmLabel="Delete"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onClose={() => setDeleting(false)}
      />
      <UnsavedChangesDialog blocker={blocker} subject="page" />
    </>
  )
}
