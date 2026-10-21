import { z } from 'zod'
import type { Vacancy, VacancyInput } from '@/api/vacancies'

export const DEPARTMENTS = [
  { value: 'deck', label: 'Deck' },
  { value: 'engine', label: 'Engine' },
  { value: 'catering_other', label: 'Catering & other' },
] as const

export const EMPLOYMENT_TYPES = [
  { value: 'contract', label: 'Contract' },
  { value: 'permanent', label: 'Permanent' },
] as const

export const VACANCY_STATUSES = [
  { value: 'draft', label: 'Draft' },
  { value: 'open', label: 'Open' },
  { value: 'closed', label: 'Closed' },
] as const

export const MAX_REQUIRED_DOCUMENTS = 20

export const vacancySchema = z.object({
  title: z.string().trim().min(1, 'The title is required.').max(200, 'The title must be at most 200 characters.'),
  slug: z.string().trim().max(80, 'The slug must be at most 80 characters.'),
  rank_id: z.string().min(1, 'Select a rank.'),
  vessel_type_id: z.string(),
  department: z.enum(['deck', 'engine', 'catering_other'], { error: 'Select a department.' }),
  location: z.string().trim().max(120, 'The location must be at most 120 characters.'),
  employment_type: z.enum(['contract', 'permanent']),
  duration: z.string().trim().max(80, 'The duration must be at most 80 characters.'),
  salary_text: z.string().trim().max(120, 'The salary text must be at most 120 characters.'),
  description_html: z.string(),
  requirements_html: z.string(),
  required_documents: z
    .array(z.string().trim().min(1).max(100, 'Each required document must be at most 100 characters.'))
    .max(MAX_REQUIRED_DOCUMENTS, `At most ${MAX_REQUIRED_DOCUMENTS} required documents are allowed.`),
  closing_date: z.string().refine((v) => v === '' || /^\d{4}-\d{2}-\d{2}$/.test(v), 'Enter a valid date.'),
  status: z.enum(['draft', 'open', 'closed']),
})

export type VacancyValues = z.infer<typeof vacancySchema>

export const emptyVacancy: VacancyValues = {
  title: '',
  slug: '',
  rank_id: '',
  vessel_type_id: '',
  department: 'deck',
  location: '',
  employment_type: 'contract',
  duration: '',
  salary_text: '',
  description_html: '',
  requirements_html: '',
  required_documents: [],
  closing_date: '',
  status: 'draft',
}

export function vacancyToValues(v: Vacancy): VacancyValues {
  return {
    title: v.title,
    slug: v.slug,
    rank_id: v.rank.id,
    vessel_type_id: v.vessel_type?.id ?? '',
    department: v.department,
    location: v.location,
    employment_type: v.employment_type,
    duration: v.duration,
    salary_text: v.salary_text,
    description_html: v.description_html,
    requirements_html: v.requirements_html,
    required_documents: v.required_documents,
    closing_date: v.closing_date ?? '',
    status: v.status,
  }
}

export function valuesToInput(values: VacancyValues): VacancyInput {
  return {
    ...values,
    title: values.title.trim(),
    slug: values.slug.trim(),
    location: values.location.trim(),
    duration: values.duration.trim(),
    salary_text: values.salary_text.trim(),
  }
}

export { previewSlug } from '@/lib/slug'

export const vacancyFormFields = [
  'title',
  'slug',
  'rank_id',
  'vessel_type_id',
  'department',
  'location',
  'employment_type',
  'duration',
  'salary_text',
  'description_html',
  'requirements_html',
  'required_documents',
  'closing_date',
  'status',
] as const
