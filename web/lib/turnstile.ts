export const TURNSTILE_SCRIPT_URL = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
const SCRIPT_ID = 'turnstile-api'

export interface TurnstileRenderOptions {
  sitekey: string
  callback: (token: string) => void
  'expired-callback': () => void
  'error-callback': () => void
}

export interface TurnstileApi {
  render: (container: HTMLElement, options: TurnstileRenderOptions) => string
  reset: (widgetId: string) => void
  remove: (widgetId: string) => void
}

declare global {
  interface Window {
    turnstile?: TurnstileApi
  }
}

export function loadTurnstile(): Promise<TurnstileApi> {
  if (window.turnstile) {
    return Promise.resolve(window.turnstile)
  }
  return new Promise<TurnstileApi>((resolve, reject) => {
    const existing = document.getElementById(SCRIPT_ID)
    const script = existing instanceof HTMLScriptElement ? existing : document.createElement('script')
    const onLoad = () => {
      if (window.turnstile) {
        resolve(window.turnstile)
      } else {
        reject(new Error('Turnstile did not initialise'))
      }
    }
    script.addEventListener('load', onLoad, { once: true })
    script.addEventListener('error', () => {
      script.remove()
      reject(new Error('Turnstile failed to load'))
    }, { once: true })
    if (!existing) {
      script.id = SCRIPT_ID
      script.src = TURNSTILE_SCRIPT_URL
      script.async = true
      script.defer = true
      document.head.append(script)
    }
  })
}
