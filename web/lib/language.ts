export const TRANSLATE_COOKIE = 'googtrans'

const ONE_YEAR_SECONDS = 31_536_000

export function isSourceLanguage(code: string, sourceCode: string): boolean {
  return code.toLowerCase() === sourceCode.toLowerCase()
}

export function targetFromCookieValue(rawValue: string | undefined, sourceCode: string): string | null {
  if (!rawValue) {
    return null
  }
  let value = rawValue
  try {
    value = decodeURIComponent(rawValue)
  } catch {
    return null
  }
  const target = /^\/([A-Za-z-]+)\/([A-Za-z-]+)$/.exec(value)?.[2]
  return target && !isSourceLanguage(target, sourceCode) ? target : null
}

export function readTargetLanguage(cookieHeader: string, sourceCode: string): string | null {
  for (const part of cookieHeader.split(';')) {
    const [name, ...rest] = part.trim().split('=')
    if (name === TRANSLATE_COOKIE) {
      return targetFromCookieValue(rest.join('='), sourceCode)
    }
  }
  return null
}

export function translateCookie(code: string, sourceCode: string): string {
  return `${TRANSLATE_COOKIE}=/${sourceCode}/${code}; path=/; max-age=${String(ONE_YEAR_SECONDS)}; SameSite=Lax`
}

export function expiredTranslateCookies(hostname: string): string[] {
  const expired = `${TRANSLATE_COOKIE}=; path=/; max-age=0; expires=Thu, 01 Jan 1970 00:00:00 GMT; SameSite=Lax`
  const domains = hostname.includes('.') ? [`; domain=${hostname}`, `; domain=.${hostname}`] : []
  return [expired, ...domains.map((domain) => `${expired}${domain}`)]
}
