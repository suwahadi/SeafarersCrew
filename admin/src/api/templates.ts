import { api } from './client'

export interface EmailTemplate {
  code: string
  name: string
  subject: string
  body_html?: string
  available_tags?: string[]
  updated_at: string
}

export interface EmailPreview {
  subject: string
  body_html: string
  unknown_tags: string[]
}

export type DocumentKind = 'sea' | 'quotation' | 'cv'

export interface DocumentTemplate {
  id: string
  name: string
  kind: DocumentKind
  paper_size: 'a4' | 'letter'
  orientation: 'portrait' | 'landscape'
  header_html?: string
  body_html?: string
  footer_html?: string
  is_default: boolean
  unknown_tags?: string[]
  updated_at: string
}

export interface DocumentTemplateInput {
  name: string
  kind: DocumentKind
  paper_size: 'a4' | 'letter'
  orientation: 'portrait' | 'landscape'
  header_html: string
  body_html: string
  footer_html: string
  is_default: boolean
}

export interface TemplateVariable {
  tag: string
  description: string
}

export interface VariableGroup {
  name: string
  kinds: DocumentKind[]
  variables: TemplateVariable[]
}

export interface VariableCatalogue {
  groups: VariableGroup[]
  cv_sections: string[]
}

export const EMAIL_TEMPLATES_KEY = ['email-templates'] as const
export const DOCUMENT_TEMPLATES_KEY = ['document-templates'] as const

export const emailTemplatesApi = {
  list: (signal?: AbortSignal) => api.get<EmailTemplate[]>('/email-templates', undefined, signal),
  get: (code: string, signal?: AbortSignal) => api.get<EmailTemplate>(`/email-templates/${code}`, undefined, signal),
  update: (code: string, subject: string, bodyHtml: string) => api.put<EmailTemplate>(`/email-templates/${code}`, { subject, body_html: bodyHtml }),
  preview: (code: string, subject: string, bodyHtml: string) =>
    api.post<EmailPreview>(`/email-templates/${code}/preview`, { subject, body_html: bodyHtml, variables: {} }),
}

export const documentTemplatesApi = {
  list: (kind?: DocumentKind, signal?: AbortSignal) => api.get<DocumentTemplate[]>('/document-templates', kind ? { kind } : undefined, signal),
  get: (id: string, signal?: AbortSignal) => api.get<DocumentTemplate>(`/document-templates/${id}`, undefined, signal),
  variables: (signal?: AbortSignal) => api.get<VariableCatalogue>('/document-templates/variables', undefined, signal),
  create: (body: DocumentTemplateInput) => api.post<DocumentTemplate>('/document-templates', body),
  update: (id: string, body: DocumentTemplateInput) => api.put<DocumentTemplate>(`/document-templates/${id}`, body),
  remove: (id: string) => api.delete(`/document-templates/${id}`),
}
