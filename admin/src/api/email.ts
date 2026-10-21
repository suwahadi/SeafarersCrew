import { api, type Query } from './client'

export type EmailStatus = 'queued' | 'sent' | 'failed'
export type EmailRelatedType = 'contract' | 'quotation' | 'crew' | 'generated_document' | 'applicant' | 'contact_message'

export interface EmailTemplate {
  code: string
  name: string
  subject: string
  body_html?: string
  available_tags?: string[]
  updated_at: string
}

export interface EmailLog {
  id: string
  template_code: string
  to: string[]
  cc: string[]
  subject: string
  related_type: EmailRelatedType | ''
  related_id: string | null
  attachment_file_ids: string[]
  brevo_message_id: string
  status: EmailStatus
  attempts: number
  error: string
  sent_at: string | null
  created_at: string
}

export interface QueuedEmail {
  log: EmailLog
  job_id: string
}

export interface SendEmailInput {
  template_code: string
  to: string[]
  cc: string[]
  subject: string
  body_html: string
  related_type?: string
  related_id?: string
  attachment_file_ids: string[]
}

export const emailApi = {
  templates: () => api.get<EmailTemplate[]>('/email-templates'),
  template: (code: string) => api.get<EmailTemplate>(`/email-templates/${code}`),
  send: (input: SendEmailInput) => api.post<QueuedEmail>('/emails/send', input),
  logs: (query: Query, signal?: AbortSignal) => api.list<EmailLog[]>('/email-logs', query, signal),
  resend: (id: string) => api.post<QueuedEmail>(`/email-logs/${id}/resend`),
}
