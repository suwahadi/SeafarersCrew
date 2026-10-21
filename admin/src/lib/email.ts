const EMAIL = /^[^\s@<>()[\],;:"]+@[^\s@<>()[\],;:"]+\.[A-Za-z]{2,}$/

export const MAX_RECIPIENTS = 20

export function splitRecipients(value: string): string[] {
  return value
    .split(/[,;\s]+/)
    .map((v) => v.trim().toLowerCase())
    .filter((v) => v !== '')
}

export interface RecipientCheck {
  valid: string[]
  invalid: string[]
}

export function checkRecipients(value: string): RecipientCheck {
  const seen = new Set<string>()
  const valid: string[] = []
  const invalid: string[] = []
  for (const r of splitRecipients(value)) {
    if (!EMAIL.test(r)) {
      invalid.push(r)
      continue
    }
    if (!seen.has(r)) {
      seen.add(r)
      valid.push(r)
    }
  }
  return { valid, invalid }
}

export function recipientError(to: string, cc: string): { to?: string; cc?: string } {
  const t = checkRecipients(to)
  const c = checkRecipients(cc)
  const out: { to?: string; cc?: string } = {}
  if (t.invalid.length > 0) out.to = `Invalid address: ${t.invalid.join(', ')}`
  else if (t.valid.length === 0) out.to = 'Add at least one recipient.'
  if (c.invalid.length > 0) out.cc = `Invalid address: ${c.invalid.join(', ')}`
  if (!out.to && !out.cc && t.valid.length + c.valid.length > MAX_RECIPIENTS) out.to = `At most ${MAX_RECIPIENTS} recipients in total.`
  return out
}
