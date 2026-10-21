import { checkRecipients, MAX_RECIPIENTS, recipientError } from './email'

describe('checkRecipients', () => {
  it('splits on commas, semicolons and whitespace, lowercases and de-duplicates', () => {
    const r = checkRecipients('A@Example.com; b@example.com, a@example.com\nc@example.org')
    expect(r.valid).toEqual(['a@example.com', 'b@example.com', 'c@example.org'])
    expect(r.invalid).toEqual([])
  })

  it('reports malformed addresses', () => {
    expect(checkRecipients('ok@example.com, nope, a@b, @x.com').invalid).toEqual(['nope', 'a@b', '@x.com'])
  })
})

describe('recipientError', () => {
  it('requires at least one valid To address', () => {
    expect(recipientError('', '').to).toBe('Add at least one recipient.')
  })

  it('names invalid To and CC addresses', () => {
    const e = recipientError('bad', 'worse')
    expect(e.to).toContain('bad')
    expect(e.cc).toContain('worse')
  })

  it('accepts valid addresses', () => {
    expect(recipientError('a@example.com', 'b@example.com')).toEqual({})
  })

  it('limits the total number of recipients', () => {
    const many = Array.from({ length: MAX_RECIPIENTS + 1 }, (_, i) => `u${i}@example.com`).join(',')
    expect(recipientError(many, '').to).toContain(String(MAX_RECIPIENTS))
  })
})
