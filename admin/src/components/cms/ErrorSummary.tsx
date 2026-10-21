import { useEffect, useRef } from 'react'

export function ErrorSummary({ lines, message }: { lines: readonly string[]; message?: string | null }) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (lines.length > 0 || message) ref.current?.focus()
  }, [lines, message])

  if (lines.length === 0 && !message) return null
  return (
    <div ref={ref} tabIndex={-1} role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-800 outline-none">
      <p className="font-medium">{message ?? `Fix ${lines.length} ${lines.length === 1 ? 'problem' : 'problems'} before saving.`}</p>
      {lines.length > 0 ? (
        <ul className="mt-1 list-disc space-y-0.5 pl-5">
          {lines.map((line) => (
            <li key={line}>{line}</li>
          ))}
        </ul>
      ) : null}
    </div>
  )
}
