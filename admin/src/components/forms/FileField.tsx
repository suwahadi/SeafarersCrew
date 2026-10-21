import { useMutation } from '@tanstack/react-query'
import { useId, useRef, useState } from 'react'
import { errorMessage } from '@/api/client'
import { filesApi, openFile, type StoredFile, type UploadContext } from '@/api/files'
import { Button } from '../ui/Button'

interface FileFieldProps {
  label: string
  context: UploadContext
  fileId: string
  onChange: (fileId: string, file?: StoredFile) => void
  accept?: string
  hint?: string
  error?: string | undefined
}

const acceptByContext: Record<UploadContext, string> = {
  photo: '.jpg,.jpeg,.png,.webp',
  cms_image: '.jpg,.jpeg,.png,.webp',
  logo: '.jpg,.jpeg,.png,.webp,.svg',
  flag: '.svg',
  crew_document: '.pdf,.jpg,.jpeg,.png,.webp',
  crew_cv: '.pdf,.jpg,.jpeg,.png,.webp,.doc,.docx',
  applicant: '.pdf,.jpg,.jpeg,.png,.webp,.doc,.docx',
  csv_import: '.csv',
}

export function FileField({ label, context, fileId, onChange, accept, hint, error }: FileFieldProps) {
  const id = useId()
  const input = useRef<HTMLInputElement>(null)
  const [name, setName] = useState('')
  const upload = useMutation({
    mutationFn: (file: File) => filesApi.upload(file, context),
    onSuccess: (f) => {
      setName(f.original_name)
      onChange(f.id, f)
    },
  })
  const view = useMutation({ mutationFn: openFile })
  const problem = upload.error ? errorMessage(upload.error) : view.error ? errorMessage(view.error) : error

  return (
    <div className="space-y-1">
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <div className="flex flex-wrap items-center gap-2">
        <input
          ref={input}
          id={id}
          type="file"
          accept={accept ?? acceptByContext[context]}
          className="sr-only"
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (file) upload.mutate(file)
            e.target.value = ''
          }}
        />
        <Button variant="secondary" size="sm" loading={upload.isPending} onClick={() => input.current?.click()}>
          {fileId ? 'Replace file' : 'Upload file'}
        </Button>
        {fileId ? (
          <>
            <Button variant="ghost" size="sm" loading={view.isPending} onClick={() => view.mutate(fileId)}>
              View
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setName('')
                onChange('')
              }}
            >
              Remove
            </Button>
            <span className="truncate text-xs text-slate-500">{name || 'File attached'}</span>
          </>
        ) : (
          <span className="text-xs text-slate-500">No file</span>
        )}
      </div>
      {problem ? (
        <p role="alert" className="text-xs text-red-600">
          {problem}
        </p>
      ) : hint ? (
        <p className="text-xs text-slate-500">{hint}</p>
      ) : null}
    </div>
  )
}
