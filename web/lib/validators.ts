export const PHONE_PATTERN = /^\+[1-9]\d{6,14}$/
export const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
export const ALLOWED_EXTENSIONS = ['pdf', 'jpg', 'jpeg', 'png', 'webp', 'doc', 'docx'] as const
export const DEFAULT_MAX_FILE_BYTES = 10 * 1024 * 1024
export const DEFAULT_MAX_DOCUMENTS = 10
export const MIN_AGE = 16
export const MAX_AGE = 80
export const MAX_MESSAGE_LENGTH = 2000
const BYTES_PER_MB = 1024 * 1024

export interface FileLike {
  name: string
  size: number
}

export function normalizePhone(value: string): string {
  return value.replace(/[\s\-().]/g, '')
}

export function isValidPhone(value: string): boolean {
  return PHONE_PATTERN.test(value)
}

export function isValidEmail(value: string): boolean {
  return value.length <= 254 && EMAIL_PATTERN.test(value)
}

function parseIsoDate(value: string): { year: number; month: number; day: number } | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  if (!match) {
    return null
  }
  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  const probe = new Date(Date.UTC(year, month - 1, day))
  if (probe.getUTCFullYear() !== year || probe.getUTCMonth() !== month - 1 || probe.getUTCDate() !== day) {
    return null
  }
  return { year, month, day }
}

export function ageOn(birthDate: string, today: Date): number | null {
  const parsed = parseIsoDate(birthDate)
  if (!parsed) {
    return null
  }
  const month = today.getMonth() + 1
  const beforeBirthday = month < parsed.month || (month === parsed.month && today.getDate() < parsed.day)
  return today.getFullYear() - parsed.year - (beforeBirthday ? 1 : 0)
}

export function birthDateError(value: string, today: Date): string | null {
  if (value === '') {
    return 'Enter your date of birth.'
  }
  const age = ageOn(value, today)
  if (age === null) {
    return 'Enter a valid date of birth.'
  }
  if (age < MIN_AGE || age > MAX_AGE) {
    return `Applicants must be between ${String(MIN_AGE)} and ${String(MAX_AGE)} years old.`
  }
  return null
}

export function isoDateYearsAgo(today: Date, years: number): string {
  const year = String(today.getFullYear() - years).padStart(4, '0')
  const month = String(today.getMonth() + 1).padStart(2, '0')
  const day = String(today.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function fileExtension(name: string): string {
  const dot = name.lastIndexOf('.')
  return dot < 0 ? '' : name.slice(dot + 1).toLowerCase()
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${String(bytes)} B`
  }
  if (bytes < BYTES_PER_MB) {
    return `${(bytes / 1024).toFixed(0)} KB`
  }
  return `${(bytes / BYTES_PER_MB).toFixed(1)} MB`
}

export function fileError(file: FileLike, maxBytes: number): string | null {
  if (!(ALLOWED_EXTENSIONS as readonly string[]).includes(fileExtension(file.name))) {
    return `${file.name}: only ${ALLOWED_EXTENSIONS.map((item) => item.toUpperCase()).join(', ')} files are allowed.`
  }
  if (file.size <= 0) {
    return `${file.name}: the file is empty.`
  }
  if (file.size > maxBytes) {
    return `${file.name}: each file may not be larger than ${String(Math.floor(maxBytes / BYTES_PER_MB))} MB.`
  }
  return null
}
