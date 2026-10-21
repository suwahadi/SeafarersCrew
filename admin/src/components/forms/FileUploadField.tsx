import { useId, useState } from 'react'
import { errorMessage } from '@/api/client'
import { filesApi, openFile, type UploadContext } from '@/api/files'
import { Button } from '../ui/Button'

interface FileUploadFieldProps {
  label: string
  value: string
  onChange: (fileId: string) => void
  context: UploadContext
  accept?: string
  hint?: string
  error?: string | undefined
}

export function FileUploadField({ label, value, onChange, context, accept = 'image/png,image/jpeg,image/webp,image/svg+xml', hint, error }: FileUploadFieldProps) {
  const id = useId()
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [name, setName] = useState<string | null>(null)

  const onFile = async (file: File | undefined) => {
    if (!file) return
    setUploading(true)
    setUploadError(null)
    try {
      const stored = await filesApi.upload(file, context)
      setName(stored.original_name)
      onChange(stored.id)
    } catch (err) {
      setUploadError(errorMessage(err))
    } finally {
      setUploading(false)
    }
  }

  const shownError = uploadError ?? error
  return (
    <div className="space-y-1">
      <span id={`${id}-label`} className="block text-sm font-medium text-slate-700">
        {label}
      </span>
      <div className="flex flex-wrap items-center gap-2">
        <label className="inline-flex h-8 cursor-pointer items-center rounded-md bg-white px-3 text-sm ring-1 ring-inset ring-slate-300 hover:bg-slate-50">
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
          {uploading ? 'Uploading…' : value ? 'Replace file' : 'Choose file'}
        </label>
        {value ? (
          <>
            <Button size="sm" variant="ghost" onClick={() => void openFile(value)}>
              {name ?? 'View current file'}
            </Button>
            <Button size="sm" variant="ghost" onClick={() => onChange('')}>
              Remove
            </Button>
          </>
        ) : (
          <span className="text-xs text-slate-500">No file</span>
        )}
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
