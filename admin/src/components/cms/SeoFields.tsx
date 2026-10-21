import { TextAreaField, TextField } from '@/components/ui/Field'
import type { UseFormRegisterReturn } from 'react-hook-form'

export const SEO_TITLE_MAX = 120
export const META_DESCRIPTION_MAX = 320

export function CharCounter({ value, max, label }: { value: string; max: number; label: string }) {
  const length = value.length
  return (
    <p className={`text-xs ${length > max ? 'font-medium text-red-600' : 'text-slate-500'}`} aria-label={`${label} length`}>
      {length}/{max}
    </p>
  )
}

interface SeoFieldsProps {
  titleField: UseFormRegisterReturn
  descriptionField: UseFormRegisterReturn
  seoTitle: string
  metaDescription: string
  titleError?: string | undefined
  descriptionError?: string | undefined
  fallbackTitle: string
  path: string
}

export function SeoFields({ titleField, descriptionField, seoTitle, metaDescription, titleError, descriptionError, fallbackTitle, path }: SeoFieldsProps) {
  const shownTitle = seoTitle.trim() || fallbackTitle || 'Untitled'
  return (
    <div className="space-y-4">
      <div>
        <TextField label="SEO title" error={titleError} hint="Shown in search results. Leave blank to use the title." {...titleField} />
        <CharCounter value={seoTitle} max={SEO_TITLE_MAX} label="SEO title" />
      </div>
      <div>
        <TextAreaField label="Meta description" rows={3} error={descriptionError} {...descriptionField} />
        <CharCounter value={metaDescription} max={META_DESCRIPTION_MAX} label="Meta description" />
      </div>
      <div className="rounded-md bg-slate-50 p-3 ring-1 ring-slate-200" aria-label="Search result preview" role="group">
        <p className="mb-1 text-xs font-medium uppercase tracking-wide text-slate-500">Search preview</p>
        <p className="truncate text-xs text-slate-600">{path}</p>
        <p className="truncate text-base text-blue-800">{shownTitle}</p>
        <p className="line-clamp-2 text-sm text-slate-600">{metaDescription.trim() || 'No description. Search engines will pick text from the page.'}</p>
      </div>
    </div>
  )
}
