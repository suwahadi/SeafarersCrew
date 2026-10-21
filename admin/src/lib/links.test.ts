import { describe, expect, it } from 'vitest'
import { buttonPairError, isHttpsUrl, isValidLink } from './links'

describe('isValidLink', () => {
  it.each(['/about', '/', 'https://example.com/x', 'mailto:crew@example.com', 'tel:+6281234567'])('accepts %s', (v) => {
    expect(isValidLink(v)).toBe(true)
  })

  it.each(['', 'about', '//evil.com', 'http://example.com', 'javascript:alert(1)', '/a b', 'mailto:', 'tel:abc'])('rejects %s', (v) => {
    expect(isValidLink(v)).toBe(false)
  })
})

describe('isHttpsUrl', () => {
  it('only accepts https addresses with a host', () => {
    expect(isHttpsUrl('https://acme.com')).toBe(true)
    expect(isHttpsUrl('http://acme.com')).toBe(false)
    expect(isHttpsUrl('https://')).toBe(false)
  })
})

describe('buttonPairError', () => {
  it('requires the label and link together', () => {
    expect(buttonPairError('Apply', '')?.field).toBe('button_link')
    expect(buttonPairError('', '/jobs')?.field).toBe('button_label')
    expect(buttonPairError('', '')).toBeNull()
    expect(buttonPairError('Apply', '/jobs')).toBeNull()
  })
})
