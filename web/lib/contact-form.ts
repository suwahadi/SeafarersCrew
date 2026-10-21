import type { FieldErrors } from '@/lib/application-form'
import { isValidEmail, isValidPhone, MAX_MESSAGE_LENGTH, normalizePhone } from '@/lib/validators'

export interface ContactValues {
  name: string
  email: string
  phone: string
  subject: string
  message: string
  website: string
}

export const CONTACT_FIELD_ORDER = ['name', 'email', 'phone', 'subject', 'message'] as const

export function validateContact(values: ContactValues): FieldErrors {
  const errors: FieldErrors = {}
  if (values.name.trim() === '') {
    errors.name = 'Enter your name.'
  }
  if (values.email.trim() === '') {
    errors.email = 'Enter your email address.'
  } else if (!isValidEmail(values.email.trim())) {
    errors.email = 'Enter a valid email address.'
  }
  if (values.phone.trim() !== '' && !isValidPhone(normalizePhone(values.phone))) {
    errors.phone = 'Use the international format with country code, for example +628123456789.'
  }
  if (values.subject.trim() === '') {
    errors.subject = 'Enter a subject.'
  } else if (values.subject.trim().length > 200) {
    errors.subject = 'The subject may not be longer than 200 characters.'
  }
  if (values.message.trim() === '') {
    errors.message = 'Enter your message.'
  } else if (values.message.length > MAX_MESSAGE_LENGTH) {
    errors.message = `Your message may not be longer than ${String(MAX_MESSAGE_LENGTH)} characters.`
  }
  return errors
}

export function buildContactBody(values: ContactValues, token: string): Record<string, string> {
  const body: Record<string, string> = {
    name: values.name.trim(),
    email: values.email.trim(),
    phone: normalizePhone(values.phone),
    subject: values.subject.trim(),
    message: values.message.trim(),
    website: values.website,
  }
  if (token !== '') {
    body['cf-turnstile-response'] = token
  }
  return body
}
