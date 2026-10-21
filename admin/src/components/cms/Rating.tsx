import { useId } from 'react'

export const RATING_VALUES = [1, 2, 3, 4, 5] as const

export function ratingText(rating: number): string {
  return `${rating} out of 5`
}

export function RatingDisplay({ rating }: { rating: number }) {
  return (
    <span className="inline-flex items-center gap-1 whitespace-nowrap">
      <span aria-hidden className="text-base leading-none tracking-tight text-amber-500">
        {RATING_VALUES.map((n) => (n <= rating ? '★' : '☆')).join('')}
      </span>
      <span className="sr-only">{ratingText(rating)}</span>
    </span>
  )
}

interface RatingInputProps {
  label: string
  value: number
  onChange: (value: number) => void
  error?: string | undefined
  required?: boolean
}

export function RatingInput({ label, value, onChange, error, required }: RatingInputProps) {
  const id = useId()
  return (
    <div className="space-y-1">
      <span id={`${id}-label`} className="block text-sm font-medium text-slate-700">
        {label}
        {required ? <span className="text-red-600"> *</span> : null}
      </span>
      <div role="radiogroup" aria-labelledby={`${id}-label`} aria-describedby={error ? `${id}-error` : undefined} className="flex gap-1">
        {RATING_VALUES.map((n) => (
          <label key={n} className="relative cursor-pointer">
            <input
              type="radio"
              name={`${id}-rating`}
              value={n}
              aria-label={n === 1 ? '1 star' : `${n} stars`}
              checked={value === n}
              onChange={() => onChange(n)}
              className="peer sr-only"
            />
            <span
              aria-hidden
              className={`block rounded px-0.5 text-2xl leading-none transition-colors duration-150 ease-out peer-focus-visible:ring-2 peer-focus-visible:ring-navy-500 motion-reduce:transition-none ${
                n <= value ? 'text-amber-500' : 'text-slate-300 hover:text-amber-300'
              }`}
            >
              ★
            </span>
          </label>
        ))}
      </div>
      {error ? (
        <p id={`${id}-error`} role="alert" className="text-xs text-red-600">
          {error}
        </p>
      ) : null}
    </div>
  )
}
