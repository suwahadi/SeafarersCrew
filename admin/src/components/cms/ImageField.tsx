import { useEffect, useId, useState } from 'react'
import { errorMessage } from '@/api/client'
import type { AdminImage } from '@/api/cmsMedia'
import { IMAGE_ACCEPT } from '@/api/cmsMedia'
import { filesApi, type UploadContext } from '@/api/files'
import { Button } from '../ui/Button'
import { Spinner } from '../ui/Spinner'

interface ImageFieldProps {
  label: string
  value: string
  onChange: (fileId: string) => void
  image?: AdminImage | null | undefined
  context?: Extract<UploadContext, 'cms_image' | 'logo' | 'flag'>
  accept?: string
  required?: boolean
  hint?: string
  error?: string | undefined
  aspect?: string
}

export function ImageField({ label, value, onChange, image, context = 'cms_image', accept = IMAGE_ACCEPT, required, hint, error, aspect = '16 / 9' }: ImageFieldProps) {
  const id = useId()
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [localPreview, setLocalPreview] = useState<string | null>(null)

  useEffect(() => {
    return () => {
      if (localPreview) URL.revokeObjectURL(localPreview)
    }
  }, [localPreview])

  const onFile = async (file: File | undefined) => {
    if (!file) return
    setUploading(true)
    setUploadError(null)
    try {
      const stored = await filesApi.upload(file, context)
      setLocalPreview(URL.createObjectURL(file))
      onChange(stored.id)
    } catch (err) {
      setUploadError(errorMessage(err))
    } finally {
      setUploading(false)
    }
  }

  const preview = value === '' ? null : (localPreview ?? (image?.file_id === value ? image.thumb_url || image.url : null))
  const shownError = uploadError ?? error

  return (
    <div className="space-y-1">
      <span id={`${id}-label`} className="block text-sm font-medium text-slate-700">
        {label}
        {required ? <span className="text-red-600"> *</span> : null}
      </span>
      <div className="flex flex-wrap items-start gap-3">
        <div className="flex w-40 items-center justify-center overflow-hidden rounded-md bg-slate-100 ring-1 ring-inset ring-slate-200" style={{ aspectRatio: aspect }}>
          {uploading ? (
            <Spinner label="Uploading image" />
          ) : preview ? (
            <img src={preview} alt="" className="h-full w-full object-contain" />
          ) : (
            <span className="text-xs text-slate-500">{value ? 'Image selected' : 'No image'}</span>
          )}
        </div>
        <div className="flex flex-col gap-2">
          <label className="inline-flex h-8 cursor-pointer items-center rounded-md bg-white px-3 text-sm ring-1 ring-inset ring-slate-300 hover:bg-slate-50 focus-within:ring-2 focus-within:ring-navy-500">
            <input
              type="file"
              accept={accept}
              className="sr-only"
              aria-labelledby={`${id}-label`}
              disabled={uploading}
              onChange={(e) => {
                void onFile(e.target.files?.[0])
                e.target.value = ''
              }}
            />
            {value ? 'Replace image' : 'Upload image'}
          </label>
          {value && !required ? (
            <Button size="sm" variant="ghost" onClick={() => onChange('')}>
              Remove
            </Button>
          ) : null}
        </div>
      </div>
      {shownError ? (
        <p role="alert" className="text-xs text-red-600">
          {shownError}
        </p>
      ) : hint ? (
        <p className="text-xs text-slate-500">{hint}</p>
      ) : null}
    </div>
  )
}
