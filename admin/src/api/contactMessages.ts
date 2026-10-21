import { api, type Query } from './client'

export type InboxStatus = 'all' | 'unread' | 'read' | 'archived'

export interface ContactMessage {
  id: string
  name: string
  email: string
  phone: string
  subject: string
  message: string
  ip_address: string
  is_read: boolean
  is_archived: boolean
  created_at: string
}

export interface ContactMessagePatch {
  is_read?: boolean
  is_archived?: boolean
}

export const UNREAD_COUNT_KEY = ['contact-messages', 'unread-count'] as const

export const contactMessagesApi = {
  list: (query: Query, signal?: AbortSignal) => api.list<ContactMessage[]>('/contact-messages', query, signal),
  unreadCount: (signal?: AbortSignal) => api.get<{ unread: number }>('/contact-messages/unread-count', undefined, signal),
  get: (id: string) => api.get<ContactMessage>(`/contact-messages/${id}`),
  patch: (id: string, patch: ContactMessagePatch) => api.patch<ContactMessage>(`/contact-messages/${id}`, patch),
}
