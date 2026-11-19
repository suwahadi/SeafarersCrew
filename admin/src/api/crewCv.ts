import { api } from './client'

export type CvChoiceKind = 'certificate' | 'document' | 'rank' | 'vessel_type'
export type CvAction = 'match' | 'create' | 'skip'
export type CvSection = 'personal' | 'documents' | 'certificates' | 'sea_service'

export interface CvChoice {
  key: string
  kind: CvChoiceKind
  text: string
  suggested_id?: string
  allow_create: boolean
}

export interface CvField {
  label: string
  value: string
  choice_key?: string
}

export interface CvRow {
  row: number
  cells: Record<string, string>
  choice_keys?: string[]
  matched_as?: string
  warnings?: string[]
}

export interface CvCellError {
  section: CvSection
  row: number
  message: string
}

export interface CvDuplicate {
  crew_id: string
  full_name: string
  reason: 'passport' | 'seaman_book' | 'name_and_birth_date'
}

export interface CvPreview {
  token: string
  full_name: string
  personal: CvField[]
  next_of_kin: CvRow[]
  documents: CvRow[]
  certificates: CvRow[]
  sea_service: CvRow[]
  not_imported: string[]
  warnings: string[]
  errors: CvCellError[]
  duplicate?: CvDuplicate
  choices: CvChoice[]
}

export interface CvDecision {
  action: CvAction
  target_id?: string
}

export interface CvCommitResult {
  crew_id: string
  full_name: string
}

export const crewCvApi = {
  validate: (file: File) => {
    const form = new FormData()
    form.set('file', file)
    return api.upload<CvPreview>('/crews/cv-import/validate', form)
  },
  commit: (token: string, decisions: Record<string, CvDecision>) => api.post<CvCommitResult>('/crews/cv-import/commit', { token, decisions }),
}
