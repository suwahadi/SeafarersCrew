import {
  birthDateError,
  fileError,
  isValidEmail,
  isValidPhone,
  MAX_MESSAGE_LENGTH,
  normalizePhone,
} from '@/lib/validators'

export type FieldErrors = Partial<Record<string, string>>

export interface ApplicationValues {
  full_name: string
  email: string
  phone: string
  date_of_birth: string
  nationality: string
  rank_applied_id: string
  years_experience: string
  message: string
  consent: boolean
  website: string
}

export interface ApplicationFiles {
  cv: File | null
  documents: readonly File[]
}

export interface ApplicationLimits {
  maxFileBytes: number
  maxDocuments: number
}

export const APPLICATION_FIELD_ORDER = [
  'full_name',
  'email',
  'phone',
  'date_of_birth',
  'nationality',
  'rank_applied_id',
  'years_experience',
  'message',
  'cv',
  'documents',
  'consent',
] as const

export function validateApplication(
  values: ApplicationValues,
  files: ApplicationFiles,
  limits: ApplicationLimits,
  today: Date,
): FieldErrors {
  const errors: FieldErrors = {}
  const years = values.years_experience.trim()

  if (values.full_name.trim() === '') {
    errors.full_name = 'Enter your full name.'
  } else if (values.full_name.trim().length > 120) {
    errors.full_name = 'Your name may not be longer than 120 characters.'
  }
  if (values.email.trim() === '') {
    errors.email = 'Enter your email address.'
  } else if (!isValidEmail(values.email.trim())) {
    errors.email = 'Enter a valid email address.'
  }
  if (values.phone.trim() === '') {
    errors.phone = 'Enter your phone number.'
  } else if (!isValidPhone(normalizePhone(values.phone))) {
    errors.phone = 'Use the international format with country code, for example +628123456789.'
  }
  const birthError = birthDateError(values.date_of_birth, today)
  if (birthError) {
    errors.date_of_birth = birthError
  }
  if (values.nationality.trim() === '') {
    errors.nationality = 'Enter your nationality.'
  }
  if (values.rank_applied_id === '') {
    errors.rank_applied_id = 'Select the rank you are applying for.'
  }
  if (years === '') {
    errors.years_experience = 'Enter your years of sea experience.'
  } else if (!/^\d+$/.test(years) || Number(years) > 60) {
    errors.years_experience = 'Enter a whole number between 0 and 60.'
  }
  if (values.message.length > MAX_MESSAGE_LENGTH) {
    errors.message = `Your message may not be longer than ${String(MAX_MESSAGE_LENGTH)} characters.`
  }
  if (!files.cv) {
    errors.cv = 'Upload your CV.'
  } else {
    const cvError = fileError(files.cv, limits.maxFileBytes)
    if (cvError) {
      errors.cv = cvError
    }
  }
  if (files.documents.length > limits.maxDocuments) {
    errors.documents = `You can upload at most ${String(limits.maxDocuments)} supporting documents.`
  } else {
    const documentError = files.documents.map((file) => fileError(file, limits.maxFileBytes)).find((item) => item)
    if (documentError) {
      errors.documents = documentError
    }
  }
  if (!values.consent) {
    errors.consent = 'You must accept the Privacy Policy to apply.'
  }
  return errors
}

export function firstInvalidField(errors: FieldErrors, order: readonly string[]): string | null {
  return order.find((name) => errors[name] !== undefined) ?? null
}

export function buildApplicationFormData(
  vacancy: string,
  values: ApplicationValues,
  files: ApplicationFiles,
  token: string,
): FormData {
  const data = new FormData()
  data.set('vacancy', vacancy)
  data.set('full_name', values.full_name.trim())
  data.set('email', values.email.trim())
  data.set('phone', normalizePhone(values.phone))
  data.set('date_of_birth', values.date_of_birth)
  data.set('nationality', values.nationality.trim())
  data.set('rank_applied_id', values.rank_applied_id)
  data.set('years_experience', values.years_experience.trim())
  data.set('message', values.message.trim())
  data.set('consent', values.consent ? 'true' : 'false')
  data.set('website', values.website)
  if (token !== '') {
    data.set('cf-turnstile-response', token)
  }
  if (files.cv) {
    data.set('cv', files.cv)
  }
  for (const document of files.documents) {
    data.append('documents', document)
  }
  return data
}
