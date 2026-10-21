import { api, type Query } from './client'

export type VacancyStatus = 'draft' | 'open' | 'closed'
export type Department = 'deck' | 'engine' | 'catering_other'
export type EmploymentType = 'contract' | 'permanent'

export interface NamedRef {
  id: string
  name: string
}

export interface Vacancy {
  id: string
  title: string
  slug: string
  rank: NamedRef
  vessel_type: NamedRef | null
  department: Department
  location: string
  employment_type: EmploymentType
  duration: string
  salary_text: string
  description_html: string
  requirements_html: string
  required_documents: string[]
  closing_date: string | null
  status: VacancyStatus
  is_expired: boolean
  published_at: string | null
  applications_count: number
  new_applications_count: number
  created_at: string
  updated_at: string
}

export interface VacancyInput {
  title: string
  slug: string
  rank_id: string
  vessel_type_id: string
  department: Department
  location: string
  employment_type: EmploymentType
  duration: string
  salary_text: string
  description_html: string
  requirements_html: string
  required_documents: string[]
  closing_date: string
  status: VacancyStatus
}

export const vacanciesApi = {
  list: (query: Query, signal?: AbortSignal) => api.list<Vacancy[]>('/vacancies', query, signal),
  get: (id: string) => api.get<Vacancy>(`/vacancies/${id}`),
  create: (input: VacancyInput) => api.post<Vacancy>('/vacancies', input),
  update: (id: string, input: VacancyInput) => api.put<Vacancy>(`/vacancies/${id}`, input),
  setStatus: (id: string, status: VacancyStatus) => api.patch<Vacancy>(`/vacancies/${id}/status`, { status }),
  remove: (id: string) => api.delete(`/vacancies/${id}`),
}
