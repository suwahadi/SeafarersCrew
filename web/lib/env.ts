const DEFAULT_API_URL = 'http://localhost:8080'
const DEFAULT_APP_URL = 'http://localhost:3000'

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

function read(name: string): string {
  return (process.env[name] ?? '').trim()
}

export function apiUrl(): string {
  return trimTrailingSlash(read('API_URL') || DEFAULT_API_URL)
}

export function publicApiUrl(): string {
  return trimTrailingSlash(read('PUBLIC_API_URL') || read('API_URL') || DEFAULT_API_URL)
}

export function appUrl(): string {
  return trimTrailingSlash(read('APP_URL') || DEFAULT_APP_URL)
}

export function revalidateSecret(): string {
  return read('REVALIDATE_SECRET')
}
