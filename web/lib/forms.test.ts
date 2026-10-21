import { describe, expect, it } from 'vitest'
import {
  buildApplicationFormData,
  firstInvalidField,
  APPLICATION_FIELD_ORDER,
  validateApplication,
  type ApplicationValues,
} from '@/lib/application-form'
import { buildContactBody, validateContact } from '@/lib/contact-form'
import { employmentTypeSchema, jobPostingJsonLd } from '@/lib/jsonld'
import { interpretFailure, limitsFromFields, retryMessage } from '@/lib/submit'
import { ageOn, birthDateError, fileError, isValidPhone, normalizePhone } from '@/lib/validators'
import type { VacancyDetail } from '@/lib/api'
import type { Site } from '@/lib/types'

const TODAY = new Date(2026, 9, 21)
const LIMITS = { maxFileBytes: 10 * 1024 * 1024, maxDocuments: 10 }

const VALID: ApplicationValues = {
  full_name: 'Budi Santoso',
  email: 'budi@example.com',
  phone: '+62 812-3456-7890',
  date_of_birth: '1990-05-17',
  nationality: 'Indonesian',
  rank_applied_id: 'rank-1',
  years_experience: '8',
  message: '',
  consent: true,
  website: '',
}

function file(name: string, size = 100): File {
  return new File([new Uint8Array(size)], name)
}

describe('phone validation', () => {
  it('accepts E.164 and rejects local formats', () => {
    expect(isValidPhone('+628123456789')).toBe(true)
    expect(isValidPhone('08123456789')).toBe(false)
    expect(isValidPhone('+0123456789')).toBe(false)
    expect(isValidPhone('+123456')).toBe(false)
  })

  it('strips separators before validating', () => {
    expect(normalizePhone('+62 (812) 345-6789')).toBe('+628123456789')
  })
})

describe('age validation', () => {
  it('computes age around the birthday', () => {
    expect(ageOn('2010-10-21', TODAY)).toBe(16)
    expect(ageOn('2010-10-22', TODAY)).toBe(15)
    expect(ageOn('2026-02-30', TODAY)).toBeNull()
  })

  it('enforces 16 to 80 years', () => {
    expect(birthDateError('2012-01-01', TODAY)).toMatch(/between 16 and 80/)
    expect(birthDateError('1940-01-01', TODAY)).toMatch(/between 16 and 80/)
    expect(birthDateError('1990-01-01', TODAY)).toBeNull()
    expect(birthDateError('', TODAY)).toMatch(/Enter/)
  })
})

describe('file validation', () => {
  it('allows the whitelisted extensions case-insensitively', () => {
    expect(fileError({ name: 'cv.PDF', size: 10 }, LIMITS.maxFileBytes)).toBeNull()
    expect(fileError({ name: 'cv.exe', size: 10 }, LIMITS.maxFileBytes)).toMatch(/only PDF/)
    expect(fileError({ name: 'noextension', size: 10 }, LIMITS.maxFileBytes)).not.toBeNull()
  })

  it('rejects empty and oversize files', () => {
    expect(fileError({ name: 'a.pdf', size: 0 }, LIMITS.maxFileBytes)).toMatch(/empty/)
    expect(fileError({ name: 'a.pdf', size: LIMITS.maxFileBytes + 1 }, LIMITS.maxFileBytes)).toMatch(/10 MB/)
  })
})

describe('validateApplication', () => {
  it('passes a complete application', () => {
    expect(validateApplication(VALID, { cv: file('cv.pdf'), documents: [] }, LIMITS, TODAY)).toEqual({})
  })

  it('reports each missing rule and orders the first invalid field', () => {
    const errors = validateApplication(
      { ...VALID, email: 'nope', phone: '0812', years_experience: '61', consent: false },
      { cv: null, documents: [] },
      LIMITS,
      TODAY,
    )
    expect(Object.keys(errors).sort()).toEqual(['consent', 'cv', 'email', 'phone', 'years_experience'])
    expect(firstInvalidField(errors, APPLICATION_FIELD_ORDER)).toBe('email')
  })

  it('limits supporting documents', () => {
    const documents = Array.from({ length: 11 }, (_, index) => file(`d${String(index)}.pdf`))
    const errors = validateApplication(VALID, { cv: file('cv.pdf'), documents }, LIMITS, TODAY)
    expect(errors.documents).toMatch(/at most 10/)
  })
})

describe('buildApplicationFormData', () => {
  it('uses the contract field names and repeats documents', () => {
    const data = buildApplicationFormData('master-ahts', VALID, { cv: file('cv.pdf'), documents: [file('a.pdf'), file('b.png')] }, 'tok')
    expect(data.get('vacancy')).toBe('master-ahts')
    expect(data.get('phone')).toBe('+6281234567890')
    expect(data.get('consent')).toBe('true')
    expect(data.get('cf-turnstile-response')).toBe('tok')
    expect(data.get('website')).toBe('')
    expect((data.get('cv') as File).name).toBe('cv.pdf')
    expect(data.getAll('documents').map((item) => (item as File).name)).toEqual(['a.pdf', 'b.png'])
  })

  it('omits the token when none was issued', () => {
    const data = buildApplicationFormData('x', VALID, { cv: file('cv.pdf'), documents: [] }, '')
    expect(data.has('cf-turnstile-response')).toBe(false)
  })
})

describe('contact form helpers', () => {
  const values = { name: 'A', email: 'a@b.co', phone: '', subject: 'S', message: 'M', website: '' }

  it('requires name, email, subject and message but keeps phone optional', () => {
    expect(validateContact(values)).toEqual({})
    expect(Object.keys(validateContact({ ...values, name: '', subject: '', message: '', phone: '123' })).sort()).toEqual([
      'message',
      'name',
      'phone',
      'subject',
    ])
  })

  it('builds the json body with the turnstile key name', () => {
    expect(buildContactBody(values, 'tok')).toMatchObject({ name: 'A', website: '', 'cf-turnstile-response': 'tok' })
  })
})

describe('interpretFailure', () => {
  const known = ['email', 'cv', 'documents']

  it('maps 422 fields and extracts server limits', () => {
    const failure = interpretFailure(
      422,
      { error: { code: 'validation_failed', message: 'x', fields: { email: 'Bad', cv: 'Each file may not be larger than 5 MB.', other: 'Odd' } } },
      null,
      known,
    )
    expect(failure.kind).toBe('fields')
    if (failure.kind === 'fields') {
      expect(failure.fields).toEqual({ email: 'Bad', cv: 'Each file may not be larger than 5 MB.' })
      expect(failure.limits.maxFileBytes).toBe(5 * 1024 * 1024)
      expect(failure.message).toBe('Odd')
    }
  })

  it('maps turnstile, conflicts, rate limits and oversize payloads', () => {
    expect(interpretFailure(422, { error: { code: 'turnstile_failed' } }, null, known).kind).toBe('turnstile')
    expect(interpretFailure(409, { error: { code: 'vacancy_closed', message: 'Closed.' } }, null, known)).toMatchObject({ kind: 'banner', code: 'vacancy_closed', message: 'Closed.' })
    expect(interpretFailure(429, null, '120', known)).toMatchObject({ kind: 'rate_limited', message: 'Too many attempts. Please try again in about 2 minutes.' })
    expect(interpretFailure(413, { error: { code: 'payload_too_large' } }, null, known).kind).toBe('too_large')
    expect(interpretFailure(500, null, null, known).kind).toBe('unavailable')
  })

  it('has friendly fallbacks for retry and limits', () => {
    expect(retryMessage(null)).toMatch(/few minutes/)
    expect(retryMessage('30')).toMatch(/1 minute\./)
    expect(limitsFromFields({ documents: 'You can upload at most 4 supporting documents.' }).maxDocuments).toBe(4)
  })
})

describe('jobPostingJsonLd', () => {
  const site = {
    name: 'Seafarers Crew',
    branding: { logo_light: null, logo_dark: null },
    organization: { legal_name: 'PT Seafarers', website: 'https://example.com', country: 'Indonesia' },
  } as unknown as Site
  const vacancy = {
    id: 'ABC123XYZ',
    title: 'Master',
    slug: 'master',
    rank: { id: 'r', name: 'Master' },
    vessel_type: null,
    department: 'deck',
    location: 'Batam',
    employment_type: 'contract',
    closing_date: '2027-03-31',
    published_at: '2026-10-21T05:45:59Z',
    description_html: '<p>Desc</p>',
    requirements_html: '<p>Req</p>',
  } as unknown as VacancyDetail

  it('sets validThrough, datePosted, employment type and hiring organization', () => {
    const node = jobPostingJsonLd(vacancy, site, 'https://site.test')
    expect(node['@type']).toBe('JobPosting')
    expect(node.validThrough).toBe('2027-03-31T23:59:59Z')
    expect(node.datePosted).toBe('2026-10-21')
    expect(node.employmentType).toBe('CONTRACTOR')
    expect(node.url).toBe('https://site.test/career/master')
    expect(node.hiringOrganization).toMatchObject({ name: 'PT Seafarers' })
    expect(node.description).toBe('<p>Desc</p><p>Req</p>')
  })

  it('omits validThrough without a closing date and maps permanent to full time', () => {
    const node = jobPostingJsonLd({ ...vacancy, closing_date: null, employment_type: 'permanent' }, site, 'https://site.test')
    expect(node.validThrough).toBeUndefined()
    expect(node.employmentType).toBe('FULL_TIME')
    expect(employmentTypeSchema('contract')).toBe('CONTRACTOR')
  })
})
