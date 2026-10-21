'use client'

import { useRef, useState, type SubmitEvent } from 'react'
import { CONTROL_CLASS, Field, fieldId, focusBanner, FormBanner, Honeypot, SubmitButton } from '@/components/forms/Field'
import { TurnstileField } from '@/components/forms/TurnstileField'
import type { FieldErrors } from '@/lib/application-form'
import { buildContactBody, CONTACT_FIELD_ORDER, validateContact, type ContactValues } from '@/lib/contact-form'
import { submitRequest } from '@/lib/submit'
import { MAX_MESSAGE_LENGTH } from '@/lib/validators'

interface ContactFormProps {
  turnstileSiteKey: string
  apiUrl: string
}

const ID_PREFIX = 'contact'
const BANNER_ID = 'contact-banner'
const BLANK: ContactValues = { name: '', email: '', phone: '', subject: '', message: '', website: '' }

export function ContactForm({ turnstileSiteKey, apiUrl }: ContactFormProps) {
  const [values, setValues] = useState<ContactValues>(BLANK)
  const [errors, setErrors] = useState<FieldErrors>({})
  const [banner, setBanner] = useState('')
  const [sending, setSending] = useState(false)
  const [sent, setSent] = useState(false)
  const [token, setToken] = useState('')
  const [resetKey, setResetKey] = useState(0)
  const [activated, setActivated] = useState(false)
  const receipt = useRef<HTMLDivElement>(null)

  function setValue<K extends keyof ContactValues>(name: K, value: ContactValues[K]): void {
    setValues((current) => ({ ...current, [name]: value }))
    setErrors((current) => (current[name] === undefined ? current : { ...current, [name]: undefined }))
  }

  function showErrors(found: FieldErrors): void {
    setErrors(found)
    const first = CONTACT_FIELD_ORDER.find((name) => found[name] !== undefined)
    if (first) {
      document.getElementById(fieldId(ID_PREFIX, first))?.focus()
    }
  }

  async function handleSubmit(event: SubmitEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault()
    if (sending) {
      return
    }
    setBanner('')
    const found = validateContact(values)
    if (Object.values(found).some((message) => message !== undefined)) {
      showErrors(found)
      return
    }
    if (turnstileSiteKey !== '' && token === '') {
      setActivated(true)
      setBanner('Please wait for the verification check to finish, then send again.')
      focusBanner(BANNER_ID)
      return
    }
    setSending(true)
    const result = await submitRequest<{ received: boolean }>(
      `${apiUrl}/api/public/v1/contact`,
      { body: JSON.stringify(buildContactBody(values, token)), headers: { 'content-type': 'application/json' } },
      CONTACT_FIELD_ORDER,
    )
    setSending(false)
    if (result.ok) {
      setSent(true)
      requestAnimationFrame(() => {
        receipt.current?.focus()
      })
      return
    }
    setToken('')
    setResetKey((current) => current + 1)
    setBanner(result.failure.message)
    if (result.failure.kind === 'fields') {
      showErrors(result.failure.fields)
    } else {
      focusBanner(BANNER_ID)
    }
  }

  if (sent) {
    return (
      <div ref={receipt} tabIndex={-1} role="status" className="rounded-lg border border-accent-text bg-surface p-6 sm:p-8">
        <h2 className="text-2xl font-bold text-primary">Thank you, we received your message</h2>
        <p className="mt-3 text-muted">
          Our team will reply to {values.email.trim()} as soon as possible.
        </p>
      </div>
    )
  }

  const common = { idPrefix: ID_PREFIX }
  return (
    <form
      noValidate
      aria-label="Contact form"
      onSubmit={(event) => {
        void handleSubmit(event)
      }}
      onFocusCapture={() => {
        setActivated(true)
      }}
      onPointerDownCapture={() => {
        setActivated(true)
      }}
      className="grid gap-5"
    >
      {banner !== '' ? <FormBanner id={BANNER_ID} message={banner} /> : null}

      <Field {...common} name="name" label="Your name" required error={errors.name}>
        {(control) => (
          <input
            {...control}
            name="name"
            type="text"
            autoComplete="name"
            maxLength={120}
            value={values.name}
            onChange={(event) => {
              setValue('name', event.target.value)
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
        label="Phone number (optional)"
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

      <Field {...common} name="subject" label="Subject" required error={errors.subject}>
        {(control) => (
          <input
            {...control}
            name="subject"
            type="text"
            maxLength={200}
            value={values.subject}
            onChange={(event) => {
              setValue('subject', event.target.value)
            }}
            className={CONTROL_CLASS}
          />
        )}
      </Field>

      <Field
        {...common}
        name="message"
        label="Message"
        required
        error={errors.message}
        extraDescribedBy={`${fieldId(ID_PREFIX, 'message')}-count`}
      >
        {(control) => (
          <>
            <textarea
              {...control}
              name="message"
              rows={6}
              maxLength={MAX_MESSAGE_LENGTH}
              value={values.message}
              onChange={(event) => {
                setValue('message', event.target.value)
              }}
              className={CONTROL_CLASS}
            />
            <p id={`${control.id}-count`} className="mt-1.5 text-right text-sm text-muted">
              {values.message.length} / {MAX_MESSAGE_LENGTH}
            </p>
          </>
        )}
      </Field>

      <Honeypot
        value={values.website}
        onChange={(website) => {
          setValue('website', website)
        }}
      />

      <TurnstileField siteKey={turnstileSiteKey} active={activated} resetKey={resetKey} onToken={setToken} />

      <div>
        <SubmitButton sending={sending} label="Send message" sendingLabel="Sending message" />
      </div>
    </form>
  )
}
