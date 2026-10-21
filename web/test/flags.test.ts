import { describe, expect, it } from 'vitest'
import { BUNDLED_FLAGS, flagSrc } from '@/lib/flags'

const noFlag = { flag: null, flag_code: '' }

describe('flagSrc', () => {
  it('prefers an uploaded flag over the bundled set', () => {
    const flag = { url: 'https://api.example.com/media/flag.svg', mime_type: 'image/svg+xml', width: 0, height: 0, alt: '', sources: [] }
    expect(flagSrc({ flag, flag_code: 'id' })).toBe('https://api.example.com/media/flag.svg')
  })

  it('falls back to the bundled flag for a known country code, ignoring case and spaces', () => {
    expect(flagSrc({ ...noFlag, flag_code: ' GB ' })).toBe('/flags/gb.svg')
  })

  it('returns null for unknown or empty codes so a text badge is shown', () => {
    expect(flagSrc({ ...noFlag, flag_code: 'zz' })).toBeNull()
    expect(flagSrc(noFlag)).toBeNull()
  })

  it('bundles one flag file for each listed code', async () => {
    const { existsSync } = await import('node:fs')
    for (const code of BUNDLED_FLAGS) expect(existsSync(`public/flags/${code}.svg`), code).toBe(true)
  })
})
