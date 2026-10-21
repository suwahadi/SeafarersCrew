import { api, type Query } from './client'
import type { NamedRef } from './vacancies'

export type Stage = 'new' | 'screening' | 'interview' | 'accepted' | 'rejected'

export type StageCounts = Record<Stage, number>

export interface Applicant {
  id: string
  vacancy: { id: string; title: string; slug: string; status: string }
  full_name: string
  email: string
  phone: string
  nationality: string
  rank_applied: NamedRef | null
  years_experience: number | null
  stage: Stage
  files_count: number
  created_at: string
}

export interface ApplicantFile {
  file_id: string
  kind: 'cv' | 'document'
  name: string
  size: number
  mime_type: string
  created_at: string
}

export interface StageLog {
  id: string
  from_stage: Stage | null
  to_stage: Stage
  note: string
  admin: { id: string; name: string } | null
  created_at: string
}

export interface ApplicantDetail extends Applicant {
  date_of_birth: string | null
  message: string
  consent_at: string | null
  ip_address: string
  updated_at: string
  files: ApplicantFile[]
  stage_logs: StageLog[]
}

export const applicantsApi = {
  list: (query: Query, signal?: AbortSignal) => api.list<Applicant[]>('/applicants', query, signal),
  stageCounts: (query: Query, signal?: AbortSignal) => api.get<StageCounts>('/applicants/stage-counts', query, signal),
  get: (id: string) => api.get<ApplicantDetail>(`/applicants/${id}`),
  setStage: (id: string, stage: Stage, note: string) => api.patch<ApplicantDetail>(`/applicants/${id}/stage`, { stage, note }),
  remove: (id: string) => api.delete(`/applicants/${id}`),
}
