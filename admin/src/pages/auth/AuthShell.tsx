import type { ReactNode } from 'react'

export function AuthShell({ title, children, footer }: { title: string; children: ReactNode; footer?: ReactNode }) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-navy-800 px-4 py-12">
      <div className="w-full max-w-sm rounded-xl bg-white p-8 shadow-xl">
        <div className="mb-6 flex items-center gap-2">
          <img src="/favicon.svg" alt="" width={32} height={32} />
          <span className="text-lg font-semibold text-navy-800">Seafarers Crew</span>
        </div>
        <h1 className="mb-4 text-xl font-semibold text-slate-900">{title}</h1>
        {children}
        {footer ? <div className="mt-6 text-center text-sm text-slate-600">{footer}</div> : null}
      </div>
    </main>
  )
}
