import { describe, expect, it } from 'vitest'
import type { ContactMessage } from '@/api/contactMessages'
import { archiveTogglePatch, openPatch, readTogglePatch, replyHref } from './inbox'

const base: ContactMessage = {
  id: 'm1',
  name: 'Ana',
  email: 'ana@example.com',
  phone: '',
  subject: 'Crew inquiry',
  message: 'Hello',
  ip_address: '203.0.113.1',
  is_read: false,
  is_archived: false,
  created_at: '2026-10-01T00:00:00Z',
}

describe('inbox patches', () => {
  it('marks an unread message as read when it is opened', () => {
    expect(openPatch(base)).toEqual({ is_read: true })
  })

  it('does not patch a message that is already read when it is opened', () => {
    expect(openPatch({ ...base, is_read: true })).toBeNull()
  })

  it('toggles the read flag', () => {
    expect(readTogglePatch(base)).toEqual({ is_read: true })
    expect(readTogglePatch({ ...base, is_read: true })).toEqual({ is_read: false })
  })

  it('toggles the archived flag', () => {
    expect(archiveTogglePatch(base)).toEqual({ is_archived: true })
    expect(archiveTogglePatch({ ...base, is_archived: true })).toEqual({ is_archived: false })
  })
})

describe('replyHref', () => {
  it('addresses the sender and prefixes the subject with Re', () => {
    expect(replyHref(base)).toBe('mailto:ana@example.com?subject=Re%3A%20Crew%20inquiry')
  })

  it('does not stack Re prefixes', () => {
    expect(replyHref({ ...base, subject: 'RE: Crew inquiry' })).toBe('mailto:ana@example.com?subject=RE%3A%20Crew%20inquiry')
  })
})
