'use client'

import { useRef, useState, type ChangeEvent, type SubmitEvent } from 'react'
import { CONTROL_CLASS, Field, fieldId, focusBanner, FormBanner, Honeypot, SubmitButton } from '@/components/forms/Field'
import { TurnstileField } from '@/components/forms/TurnstileField'
import { ButtonLink } from '@/components/ui/button'
import {
  APPLICATION_FIELD_ORDER,
  buildApplicationFormData,
  firstInvalidField,
  validateApplication,
  type ApplicationLimits,
  type ApplicationValues,
  type FieldErrors,
} from '@/lib/application-form'
import { submitRequest } from '@/lib/submit'
import {
  ALLOWED_EXTENSIONS,
  DEFAULT_MAX_DOCUMENTS,
  DEFAULT_MAX_FILE_BYTES,
  fileError,
  formatBytes,
  isoDateYearsAgo,
  MAX_AGE,
  MAX_MESSAGE_LENGTH,
  MIN_AGE,
} from '@/lib/validators'

export interface RankOption {
  id: string
  name: string
}

export type ApplicationStatus = 'editing' | 'sending' | 'sent'

export interface ApplicationFormProps {
  vacancySlug: string
  ranks: readonly RankOption[]
  defaultRankId: string
  turnstileSiteKey: string
  apiUrl: string
  onStatusChange?: (status: ApplicationStatus) => void
}

const ID_PREFIX = 'application'
const BANNER_ID = 'application-banner'
const ACCEPT = ALLOWED_EXTENSIONS.map((extension) => `.${extension}`).join(',')

const SERVER_FIELDS: readonly string[] = [...APPLICATION_FIELD_ORDER]

const BLANK_VALUES: Omit<ApplicationValues, 'rank_applied_id'> = {
  full_name: '',
  email: '',
  phone: '',
  date_of_birth: '',
  nationality: '',
  years_experience: '',
  message: '',
  consent: false,
  website: '',
}

function sameFile(a: File, b: File): boolean {
  return a.name === b.name && a.size === b.size && a.lastModified === b.lastModified
}

function focusField(name: string): void {
  document.getElementById(fieldId(ID_PREFIX, name))?.focus()
}

export function ApplicationForm({ vacancySlug, ranks, defaultRankId, turnstileSiteKey, apiUrl, onStatusChange }: ApplicationFormProps) {
  const [values, setValues] = useState<ApplicationValues>({ ...BLANK_VALUES, rank_applied_id: defaultRankId })
  const [cv, setCv] = useState<File | null>(null)
  const [documents, setDocuments] = useState<File[]>([])
  const [errors, setErrors] = useState<FieldErrors>({})
  const [banner, setBanner] = useState('')
  const [sending, setSending] = useState(false)
  const [applicationId, setApplicationId] = useState('')
  const [token, setToken] = useState('')
  const [resetKey, setResetKey] = useState(0)
  const [activated, setActivated] = useState(false)
  const [limits, setLimits] = useState<ApplicationLimits>({
    maxFileBytes: DEFAULT_MAX_FILE_BYTES,
    maxDocuments: DEFAULT_MAX_DOCUMENTS,
  })
  const receipt = useRef<HTMLDivElement>(null)

  function setValue<K extends keyof ApplicationValues>(name: K, value: ApplicationValues[K]): void {
    setValues((current) => ({ ...current, [name]: value }))
    setErrors((current) => (current[name] === undefined ? current : { ...current, [name]: undefined }))
  }

  function handleCv(event: ChangeEvent<HTMLInputElement>): void {
    const file = event.target.files?.[0] ?? null
    if (!file) {
      setCv(null)
      return
    }
    const problem = fileError(file, limits.maxFileBytes)
    if (problem) {
      event.target.value = ''
      setCv(null)
      setErrors((current) => ({ ...current, cv: problem }))
      return
    }
    setCv(file)
    setErrors((current) => ({ ...current, cv: undefined }))
  }

  function handleDocuments(event: ChangeEvent<HTMLInputElement>): void {
    const picked = [...(event.target.files ?? [])]
    event.target.value = ''
    let next = documents
    let problem: string | undefined
    for (const file of picked) {
      const invalid = fileError(file, limits.maxFileBytes)
      if (invalid) {
        problem ??= invalid
      } else if (next.some((existing) => sameFile(existing, file))) {
        problem ??= `${file.name}: this file is already in the list.`
      } else if (next.length >= limits.maxDocuments) {
        problem ??= `You can upload at most ${String(limits.maxDocuments)} supporting documents.`
      } else {
        next = [...next, file]
      }
    }
    setDocuments(next)
    setErrors((current) => ({ ...current, documents: problem }))
  }

  function removeDocument(index: number): void {
    setDocuments((current) => current.filter((_, position) => position !== index))
    setErrors((current) => ({ ...current, documents: undefined }))
  }

  function showErrors(found: FieldErrors): void {
    setErrors(found)
    const first = firstInvalidField(found, APPLICATION_FIELD_ORDER)
    if (first) {
      focusField(first)
    }
  }

  async function handleSubmit(event: SubmitEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    if (sending) {
      return
    }
    setBanner('')
    const found = validateApplication(values, { cv, documents }, limits, new Date())
    if (Object.values(found).some((message) => message !== undefined)) {
      showErrors(found)
      return
    }
    if (turnstileSiteKey !== '' && token === '') {
      setActivated(true)
      setBanner('Please wait for the verification check to finish, then submit again.')
      focusBanner(BANNER_ID)
      return
    }
    setSending(true)
    onStatusChange?.('sending')
    const result = await submitRequest<{ application_id: string }>(
      `${apiUrl}/api/public/v1/applications`,
      { body: buildApplicationFormData(vacancySlug, values, { cv, documents }, token) },
      SERVER_FIELDS,
    )
    setSending(false)
    onStatusChange?.(result.ok ? 'sent' : 'editing')
    if (result.ok) {
      setApplicationId(result.data.application_id)
      requestAnimationFrame(() => {
        receipt.current?.focus()
        receipt.current?.scrollIntoView({ block: 'center' })
      })
      return
    }
    setToken('')
    setResetKey((current) => current + 1)
    const { failure } = result
    setBanner(failure.message)
    if (failure.kind === 'fields') {
      setLimits((current) => ({ ...current, ...failure.limits }))
      showErrors(failure.fields)
    } else {
      focusBanner(BANNER_ID)
    }
  }

  if (applicationId !== '') {
    return (
      <div
        ref={receipt}
        tabIndex={-1}
        role="status"
        className="rounded-lg border border-accent-text bg-surface p-6 text-center sm:p-10"
      >
        <h3 className="text-2xl font-bold text-primary">Thank you, your application has been received</h3>
        <p className="mt-4 text-sm font-semibold uppercase tracking-wider text-muted">Your Application ID</p>
        <p translate="no" className="mt-1 text-4xl font-bold tracking-widest text-primary" data-testid="application-id">
          {applicationId}
        </p>
        <p className="mx-auto mt-4 max-w-xl text-muted">
          A confirmation email was sent to {values.email.trim()}. Please keep your Application ID and quote it when you
          contact us.
        </p>
        <div className="mt-6 flex flex-wrap justify-center gap-3">
          <ButtonLink href="/career">Browse more vacancies</ButtonLink>
          <ButtonLink href="/" variant="outline">
            Back to home
          </ButtonLink>
        </div>
      </div>
    )
  }

  const today = new Date()
  const messageLength = values.message.length
  const common = { idPrefix: ID_PREFIX }

  return (
    <form
      noValidate
      onSubmit={(event) => {
        void handleSubmit(event)
      }}
      onFocusCapture={() => {
        setActivated(true)
      }}
      onPointerDownCapture={() => {
        setActivated(true)
      }}
      aria-label="Job application"
      className="grid gap-5 sm:grid-cols-2"
    >
      {banner !== '' ? (
        <div className="sm:col-span-2">
          <FormBanner id={BANNER_ID} message={banner} />
        </div>
      ) : null}

      <Field {...common} name="full_name" label="Full name" required error={errors.full_name}>
        {(control) => (
          <input
            {...control}
            name="full_name"
            type="text"
            autoComplete="name"
            maxLength={120}
            value={values.full_name}
            onChange={(event) => {
              setValue('full_name', event.target.value)
            }}
            className={CONTROL_CLASS}
          />
        )}
      </Field>

      <Field {...common} name="email" label="Email address" required error={errors.email}>
        {(control) => (
          <input
            {...control}
            name="email"
            type="email"
            autoComplete="email"
            inputMode="email"
            value={values.email}
            onChange={(event) => {
              setValue('email', event.target.value)
            }}
            className={CONTROL_CLASS}
          />
        )}
      </Field>

      <Field
        {...common}
        name="phone"
        label="Phone number"
        required
        error={errors.phone}
        hint="International format with country code."
      >
        {(control) => (
          <input
            {...control}
            name="phone"
            type="tel"
            autoComplete="tel"
            inputMode="tel"
            placeholder="+62 812 3456 7890"
            value={values.phone}
            onChange={(event) => {
              setValue('phone', event.target.value)
            }}
            className={CONTROL_CLASS}
          />
        )}
      </Field>

      <Field {...common} name="date_of_birth" label="Date of birth" required error={errors.date_of_birth}>
        {(control) => (
          <input
            {...control}
            name="date_of_birth"
            type="date"
            autoComplete="bday"
            min={isoDateYearsAgo(today, MAX_AGE)}
            max={isoDateYearsAgo(today, MIN_AGE)}
            value={values.date_of_birth}
            onChange={(event) => {
              setValue('date_of_birth', event.target.value)
            }}
            className={CONTROL_CLASS}
          />
        )}
      </Field>

      <Field {...common} name="nationality" label="Nationality" required error={errors.nationality}>
        {(control) => (
          <input
            {...control}
            name="nationality"
            type="text"
            autoComplete="off"
            maxLength={80}
            value={values.nationality}
            onChange={(event) => {
              setValue('nationality', event.target.value)
            }}
            className={CONTROL_CLASS}
          />
        )}
      </Field>

      <Field {...common} name="rank_applied_id" label="Rank applied for" required error={errors.rank_applied_id}>
        {(control) => (
          <select
            {...control}
            name="rank_applied_id"
            value={values.rank_applied_id}
            onChange={(event) => {
              setValue('rank_applied_id', event.target.value)
            }}
            className={CONTROL_CLASS}
          >
            <option value="">Select a rank</option>
            {ranks.map((rank) => (
              <option key={rank.id} value={rank.id}>
                {rank.name}
              </option>
            ))}
          </select>
        )}
      </Field>

      <Field
        {...common}
        name="years_experience"
        label="Years of sea experience"
        required
        error={errors.years_experience}
        className="sm:col-span-2"
      >
        {(control) => (
          <input
            {...control}
            name="years_experience"
            type="number"
            inputMode="numeric"
            min={0}
            max={60}
            step={1}
            value={values.years_experience}
            onChange={(event) => {
              setValue('years_experience', event.target.value)
            }}
            className={`${CONTROL_CLASS} sm:max-w-48`}
          />
        )}
      </Field>

      <Field
        {...common}
        name="message"
        label="Cover message (optional)"
        error={errors.message}
        extraDescribedBy={`${fieldId(ID_PREFIX, 'message')}-count`}
        className="sm:col-span-2"
      >
        {(control) => (
          <>
            <textarea
              {...control}
              name="message"
              rows={5}
              maxLength={MAX_MESSAGE_LENGTH}
              value={values.message}
              onChange={(event) => {
                setValue('message', event.target.value)
              }}
              className={CONTROL_CLASS}
            />
            <p id={`${control.id}-count`} className="mt-1.5 text-right text-sm text-muted">
              {messageLength} / {MAX_MESSAGE_LENGTH}
            </p>
          </>
        )}
      </Field>

      <Field
        {...common}
        name="cv"
        label="CV"
        required
        error={errors.cv}
        hint={`One file. PDF, JPG, PNG, WebP, DOC or DOCX, up to ${String(Math.floor(limits.maxFileBytes / (1024 * 1024)))} MB.`}
        className="sm:col-span-2"
      >
        {(control) => (
          <input
            {...control}
            name="cv"
            type="file"
            accept={ACCEPT}
            onChange={handleCv}
            className={`${CONTROL_CLASS} file:mr-3 file:rounded file:border-0 file:bg-surface file:px-3 file:py-1.5 file:text-sm file:font-semibold file:text-primary`}
          />
        )}
      </Field>

      <Field
        {...common}
        name="documents"
        label="Supporting documents (optional)"
        error={errors.documents}
        hint={`Certificates, passport pages and similar. Up to ${String(limits.maxDocuments)} files, ${String(Math.floor(limits.maxFileBytes / (1024 * 1024)))} MB each.`}
        className="sm:col-span-2"
      >
        {(control) => (
          <>
            <input
              {...control}
              name="documents"
              type="file"
              multiple
              accept={ACCEPT}
              onChange={handleDocuments}
              className={`${CONTROL_CLASS} file:mr-3 file:rounded file:border-0 file:bg-surface file:px-3 file:py-1.5 file:text-sm file:font-semibold file:text-primary`}
            />
            {documents.length > 0 ? (
              <ul className="mt-3 divide-y divide-line rounded-md border border-line" aria-label="Selected documents">
                {documents.map((file, index) => (
                  <li key={`${file.name}-${String(file.size)}-${String(file.lastModified)}`} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                    <span className="min-w-0 truncate">
                      {file.name} <span className="text-muted">({formatBytes(file.size)})</span>
                    </span>
                    <button
                      type="button"
                      onClick={() => {
                        removeDocument(index)
                      }}
                      aria-label={`Remove ${file.name}`}
                      className="shrink-0 rounded px-2 py-1 font-semibold text-red-700 underline-offset-2 hover:underline"
                    >
                      Remove
                    </button>
                  </li>
                ))}
              </ul>
            ) : null}
          </>
        )}
      </Field>

      <div className="sm:col-span-2">
        <Field {...common} name="consent" error={errors.consent}>
          {(control) => (
            <div className="flex items-start gap-3">
              <input
                {...control}
                name="consent"
                type="checkbox"
                checked={values.consent}
                onChange={(event) => {
                  setValue('consent', event.target.checked)
                }}
                className="mt-1 size-5 shrink-0 accent-[var(--color-primary)]"
              />
              <label htmlFor={control.id} className="text-sm">
                I have read the{' '}
                <a
                  href="/privacy-policy"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-semibold text-accent-text underline underline-offset-2"
                >
                  Privacy Policy
                </a>{' '}
                and consent to the processing of my data
              </label>
            </div>
          )}
        </Field>
      </div>

      <Honeypot
        value={values.website}
        onChange={(website) => {
          setValue('website', website)
        }}
      />

      <div className="sm:col-span-2">
        <TurnstileField siteKey={turnstileSiteKey} active={activated} resetKey={resetKey} onToken={setToken} />
      </div>

      <div className="sm:col-span-2">
        <SubmitButton sending={sending} label="Submit application" sendingLabel="Sending application" />
      </div>
    </form>
  )
}
