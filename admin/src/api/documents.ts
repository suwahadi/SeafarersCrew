import { api } from './client'

export type DocumentKind = 'sea' | 'quotation' | 'cv'
export type RelatedType = 'contract' | 'quotation' | 'crew'

export const RELATED_BY_KIND: Record<DocumentKind, RelatedType> = { sea: 'contract', quotation: 'quotation', cv: 'crew' }
export const KIND_LABELS: Record<DocumentKind, string> = { sea: 'Seafarer Employment Agreement', quotation: 'Quotation', cv: 'Crew CV' }

export interface DocumentTemplateSummary {
  id: string
  name: string
  kind: DocumentKind
  paper_size: string
  orientation: string
  is_default: boolean
  updated_at: string
}

export interface TemplateVariables {
  groups: unknown
  cv_sections: string[]
}

export interface DocumentPreview {
  html: string
  unknown_tags: string[]
}

export interface GeneratedDocument {
  id: string
  kind: DocumentKind
  template_id: string | null
  template_name: string
  related_type: RelatedType
  file_id: string
  file_size: number
  created_at: string
}

export interface RenderInput {
  kind: DocumentKind
  template_id?: string
  related_id: string
  hide?: string[]
}

export const documentsApi = {
  templates: (kind: DocumentKind) => api.get<DocumentTemplateSummary[]>('/document-templates', { kind }),
  variables: () => api.get<TemplateVariables>('/document-templates/variables'),
  preview: (input: RenderInput) => api.post<DocumentPreview>('/documents/preview', input),
  generate: (input: RenderInput) => api.post<GeneratedDocument>('/documents/generate', input),
  history: (relatedType?: RelatedType, relatedId?: string) =>
    api.get<GeneratedDocument[]>('/generated-documents', { related_type: relatedType, related_id: relatedId }),
}
