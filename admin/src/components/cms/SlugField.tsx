import { forwardRef, type InputHTMLAttributes } from 'react'
import { Button } from '@/components/ui/Button'
import { TextField } from '@/components/ui/Field'
import { makeSlug, previewSlug } from '@/lib/slug'

interface SlugFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  title: string
  slug: string
  existingSlug: string | undefined
  prefix: string
  error?: string | undefined
  onAuto: (slug: string) => void
  hint?: string
}

export const SlugField = forwardRef<HTMLInputElement, SlugFieldProps>(function SlugField({ title, slug, existingSlug, prefix, error, onAuto, hint, ...input }, ref) {
  const preview = previewSlug(title, slug, existingSlug)
  const defaultHint = existingSlug ? 'Leave blank to keep the current slug. Changing it redirects the old address.' : 'Leave blank to generate it from the title.'
  return (
    <div className="space-y-1">
      <div className="flex items-end gap-2">
        <div className="flex-1">
          <TextField label="Slug" ref={ref} error={error} hint={hint ?? defaultHint} {...input} />
        </div>
        <Button variant="secondary" size="sm" className="mb-5" disabled={input.disabled} onClick={() => onAuto(makeSlug(title))}>
          Auto from title
        </Button>
      </div>
      <p className="text-xs text-slate-600">
        Public address:{' '}
        <code data-testid="slug-preview" className="rounded bg-slate-100 px-1 py-0.5">
          {preview === '' ? `${prefix}…` : `${prefix}${preview}`}
        </code>
      </p>
    </div>
  )
})
