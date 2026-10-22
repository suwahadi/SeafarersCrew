import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'

type ToastTone = 'success' | 'error' | 'info'

interface Toast {
  id: number
  tone: ToastTone
  message: string
}

interface ToastApi {
  success: (message: string) => void
  error: (message: string) => void
  info: (message: string) => void
}

const ToastContext = createContext<ToastApi | null>(null)

const toneClass: Record<ToastTone, string> = {
  success: 'bg-green-700',
  error: 'bg-red-700',
  info: 'bg-navy-800',
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const push = useCallback((tone: ToastTone, message: string) => {
    const id = Date.now() + Math.random()
    setToasts((t) => [...t, { id, tone, message }])
    window.setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 5000)
  }, [])

  const value = useMemo<ToastApi>(
    () => ({ success: (m) => push('success', m), error: (m) => push('error', m), info: (m) => push('info', m) }),
    [push],
  )

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div aria-live="polite" className="pointer-events-none fixed inset-x-4 bottom-4 z-50 flex flex-col gap-2 sm:left-auto sm:w-80">
        {toasts.map((t) => (
          <div key={t.id} role={t.tone === 'error' ? 'alert' : 'status'} className={`pointer-events-auto rounded-md px-4 py-3 text-sm text-white shadow-lg ${toneClass[t.tone]}`}>
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used inside ToastProvider')
  return ctx
}
