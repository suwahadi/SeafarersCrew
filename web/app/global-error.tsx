'use client'

import './globals.css'

export default function GlobalError({ reset }: { reset: () => void }) {
  return (
    <html lang="en">
      <body className="flex min-h-dvh flex-col items-center justify-center gap-4 px-6 text-center">
        <h1 className="text-3xl font-bold text-primary">Service temporarily unavailable</h1>
        <p className="max-w-md text-muted">We are having trouble loading the website. Please try again shortly.</p>
        <button type="button" onClick={reset} className="rounded-md bg-primary px-5 py-3 text-sm font-semibold text-on-primary">
          Try again
        </button>
      </body>
    </html>
  )
}
