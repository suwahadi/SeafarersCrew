import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { Link, useBlocker, useNavigate, useParams } from 'react-router'
import { errorMessage } from '@/api/client'
import { vacanciesApi, type Vacancy, type VacancyStatus } from '@/api/vacancies'
import { RichTextEditor } from '@/components/editor/RichTextEditor'
import { TagInput } from '@/components/forms/TagInput'
import { Button } from '@/components/ui/Button'
import { SelectField, TextField } from '@/components/ui/Field'
import { Card, PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { useOptions } from '@/hooks/useOptions'
import { applyServerErrors } from '@/lib/forms'
import { makeSlug } from '@/lib/slug'
import {
  DEPARTMENTS,
  EMPLOYMENT_TYPES,
  MAX_REQUIRED_DOCUMENTS,
  VACANCY_STATUSES,
  emptyVacancy,
  previewSlug,
  vacancySchema,
  vacancyFormFields,
  vacancyToValues,
  valuesToInput,
  type VacancyValues,
} from './vacancyForm'
import { VacancyStatusBadge } from './VacancyStatusBadge'

export const vacancyKeys = {
  detail: (id: string | undefined) => ['vacancy', id] as const,
}

export default function VacancyFormPage() {
  const { id } = useParams()
  const vacancy = useQuery({ queryKey: vacancyKeys.detail(id), queryFn: () => vacanciesApi.get(id ?? ''), enabled: id !== undefined })
  if (id !== undefined && vacancy.isPending) {
    return (
      <div className="flex justify-center p-12 text-navy-800">
        <Spinner size="lg" label="Loading vacancy" />
      </div>
    )
  }
  if (vacancy.isError) return <p role="alert" className="text-sm text-red-700">{errorMessage(vacancy.error)}</p>
  return <VacancyForm key={vacancy.data?.updated_at ?? 'new'} vacancy={vacancy.data} />
}

export function VacancyForm({ vacancy }: { vacancy: Vacancy | undefined }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const { options } = useOptions()
  const [formError, setFormError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)
  const leaving = useRef(false)

  const form = useForm<VacancyValues>({
    resolver: zodResolver(vacancySchema),
    defaultValues: vacancy ? vacancyToValues(vacancy) : emptyVacancy,
  })
  const { register, control, handleSubmit, setError, setValue, getValues, formState } = form
  const [title, slug] = useWatch({ control, name: ['title', 'slug'] })
  const dirty = formState.isDirty

  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && !leaving.current && currentLocation.pathname !== nextLocation.pathname)

  useEffect(() => {
    if (!dirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [dirty])

  const invalidateLists = () => {
    void qc.invalidateQueries({ queryKey: ['grid', 'vacancies'] })
    void qc.invalidateQueries({ queryKey: ['vacancy-options'] })
  }

  const save = useMutation({
    mutationFn: (values: VacancyValues) => {
      const input = valuesToInput(values)
      return vacancy ? vacanciesApi.update(vacancy.id, input) : vacanciesApi.create(input)
    },
    onSuccess: (saved) => {
      toast.success(vacancy ? 'Vacancy saved.' : `Vacancy ${saved.id} created.`)
      qc.setQueryData(vacancyKeys.detail(saved.id), saved)
      invalidateLists()
      if (!vacancy) {
        leaving.current = true
        void navigate(`/vacancies/${saved.id}`, { replace: true })
      }
    },
    onError: (err) => setFormError(applyServerErrors(err, setError, vacancyFormFields)),
  })

  const changeStatus = useMutation({
    mutationFn: (status: VacancyStatus) => vacanciesApi.setStatus(vacancy?.id ?? '', status),
    onSuccess: (saved) => {
      toast.success(saved.status === 'open' ? 'Vacancy opened.' : 'Vacancy closed.')
      qc.setQueryData(vacancyKeys.detail(saved.id), saved)
      invalidateLists()
    },
    onError: (err) => setFormError(errorMessage(err)),
  })

  const remove = useMutation({
    mutationFn: () => vacanciesApi.remove(vacancy?.id ?? ''),
    onSuccess: () => {
      leaving.current = true
      qc.removeQueries({ queryKey: vacancyKeys.detail(vacancy?.id) })
      invalidateLists()
      toast.success('Vacancy deleted.')
      void navigate('/vacancies', { replace: true })
    },
    onError: (err) => {
      toast.error(errorMessage(err))
      setDeleting(false)
    },
  })

  const onSubmit = handleSubmit((values) => {
    setFormError(null)
    save.mutate(values)
  })

  const publicSlug = previewSlug(title, slug, vacancy?.slug)
  const slugHint = vacancy
    ? 'Leave blank to keep the current slug. Changing it redirects the old address.'
    : 'Leave blank to generate it from the title.'

  const onRankChange = (rankId: string) => {
    const rank = options.ranks.find((r) => r.id === rankId)
    if (rank && !formState.dirtyFields.department) setValue('department', rank.department, { shouldDirty: true })
  }

  return (
    <>
      <PageHeader
        title={vacancy ? `Vacancy ${vacancy.id}` : 'New vacancy'}
        description={vacancy ? vacancy.title : 'Publish an opening on the career page.'}
        actions={
          <>
            {vacancy ? <VacancyStatusBadge vacancy={vacancy} /> : null}
            {vacancy ? (
              <Link to={`/applicants?vacancy=${vacancy.id}`} className="text-sm text-navy-700 hover:underline">
                {vacancy.applications_count} application{vacancy.applications_count === 1 ? '' : 's'}
                {vacancy.new_applications_count > 0 ? ` (${vacancy.new_applications_count} new)` : ''}
              </Link>
            ) : null}
            <Link to="/vacancies" className="text-sm text-navy-700 hover:underline">
              Back to vacancies
            </Link>
          </>
        }
      />
      <form onSubmit={onSubmit} noValidate className="space-y-6">
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <Card title="Position">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="md:col-span-2">
              <TextField label="Title" required error={formState.errors.title?.message} {...register('title')} />
            </div>
            <div className="space-y-1">
              <div className="flex items-end gap-2">
                <div className="flex-1">
                  <TextField label="Slug" error={formState.errors.slug?.message} hint={slugHint} {...register('slug')} />
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  className="mb-5"
                  onClick={() => setValue('slug', makeSlug(getValues('title')), { shouldDirty: true })}
                >
                  Auto from title
                </Button>
              </div>
              <p className="text-xs text-slate-600">
                Public address:{' '}
                <code data-testid="slug-preview" className="rounded bg-slate-100 px-1 py-0.5">
                  {publicSlug === '' ? '/career/…' : `/career/${publicSlug}`}
                </code>
              </p>
            </div>
            <SelectField
              label="Rank"
              required
              placeholder="Select a rank"
              options={options.ranks.map((r) => ({ value: r.id, label: r.name }))}
              error={formState.errors.rank_id?.message}
              {...register('rank_id', { onChange: (e: { target: { value: string } }) => onRankChange(e.target.value) })}
            />
            <SelectField
              label="Vessel type"
              placeholder="Any vessel type"
              options={options.vessel_types.map((t) => ({ value: t.id, label: t.name }))}
              error={formState.errors.vessel_type_id?.message}
              {...register('vessel_type_id')}
            />
            <SelectField label="Department" required options={[...DEPARTMENTS]} error={formState.errors.department?.message} {...register('department')} />
            <SelectField label="Employment type" required options={[...EMPLOYMENT_TYPES]} error={formState.errors.employment_type?.message} {...register('employment_type')} />
            <TextField label="Location" error={formState.errors.location?.message} {...register('location')} />
            <TextField label="Duration" hint="For example 9 months." error={formState.errors.duration?.message} {...register('duration')} />
            <TextField label="Salary" hint="Shown as free text, for example USD 1,800 – 2,200 per month." error={formState.errors.salary_text?.message} {...register('salary_text')} />
          </div>
        </Card>
        <Card title="Description">
          <div className="space-y-4">
            <Controller
              control={control}
              name="description_html"
              render={({ field, fieldState }) => (
                <RichTextEditor label="Description" value={field.value} onChange={field.onChange} withLayout={false} error={fieldState.error?.message} />
              )}
            />
            <Controller
              control={control}
              name="requirements_html"
              render={({ field, fieldState }) => (
                <RichTextEditor label="Requirements" value={field.value} onChange={field.onChange} withLayout={false} error={fieldState.error?.message} />
              )}
            />
            <Controller
              control={control}
              name="required_documents"
              render={({ field, fieldState }) => (
                <TagInput
                  label="Required documents"
                  value={field.value}
                  onChange={field.onChange}
                  placeholder="Type a document and press Enter"
                  hint={`Applicants are told to bring these. Up to ${MAX_REQUIRED_DOCUMENTS}.`}
                  error={fieldState.error?.message ?? formState.errors.required_documents?.root?.message}
                  validate={(item) => {
                    if (item.length > 100) return 'Each required document must be at most 100 characters.'
                    if (!field.value.includes(item) && field.value.length >= MAX_REQUIRED_DOCUMENTS) return `At most ${MAX_REQUIRED_DOCUMENTS} required documents are allowed.`
                    return null
                  }}
                />
              )}
            />
          </div>
        </Card>
        <Card title="Publishing">
          <div className="grid gap-4 md:grid-cols-2">
            <TextField label="Closing date" type="date" error={formState.errors.closing_date?.message} hint="An open vacancy needs a closing date of today or later." {...register('closing_date')} />
            <SelectField label="Status" required options={[...VACANCY_STATUSES]} error={formState.errors.status?.message} {...register('status')} />
          </div>
        </Card>
        <div className="flex flex-wrap items-center gap-2">
          <Button type="submit" loading={save.isPending}>
            {vacancy ? 'Save changes' : 'Create vacancy'}
          </Button>
          {vacancy ? (
            <>
              {vacancy.status !== 'open' ? (
                <Button variant="secondary" disabled={dirty} loading={changeStatus.isPending} onClick={() => changeStatus.mutate('open')}>
                  Open vacancy
                </Button>
              ) : null}
              {vacancy.status === 'open' ? (
                <Button variant="secondary" disabled={dirty} loading={changeStatus.isPending} onClick={() => changeStatus.mutate('closed')}>
                  Close vacancy
                </Button>
              ) : null}
              {dirty ? <span className="text-xs text-slate-500">Save your changes to use the quick status actions.</span> : null}
              <Button variant="danger" className="ml-auto" onClick={() => setDeleting(true)}>
                Delete
              </Button>
            </>
          ) : null}
        </div>
      </form>
      <ConfirmDialog
        open={deleting}
        title="Delete vacancy?"
        message="The vacancy disappears from the career page. Its applicants stay in the archive."
        confirmLabel="Delete"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onClose={() => setDeleting(false)}
      />
      <ConfirmDialog
        open={blocker.state === 'blocked'}
        title="Discard unsaved changes?"
        message="You have unsaved changes on this vacancy. Leaving now will discard them."
        confirmLabel="Discard and leave"
        danger
        onConfirm={() => blocker.state === 'blocked' && blocker.proceed()}
        onClose={() => blocker.state === 'blocked' && blocker.reset()}
      />
    </>
  )
}
