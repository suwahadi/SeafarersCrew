import type { ContactMessage, ContactMessagePatch } from '@/api/contactMessages'

export function openPatch(message: ContactMessage): ContactMessagePatch | null {
  return message.is_read ? null : { is_read: true }
}

export function readTogglePatch(message: ContactMessage): ContactMessagePatch {
  return { is_read: !message.is_read }
}

export function archiveTogglePatch(message: ContactMessage): ContactMessagePatch {
  return { is_archived: !message.is_archived }
}

export function replyHref(message: ContactMessage): string {
  const subject = /^re:/i.test(message.subject.trim()) ? message.subject.trim() : `Re: ${message.subject.trim()}`
  return `mailto:${message.email}?subject=${encodeURIComponent(subject)}`
}
