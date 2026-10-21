import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type SubmitEventHandler, type ReactNode } from 'react'
import type { FieldValues, Path, UseFormSetError } from 'react-hook-form'
import { cmsApi, cmsKey, type CmsBase, type CmsSegment } from '@/api/cms'
import { Button } from '@/components/ui/Button'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors } from '@/lib/forms'

interface SaveOptions<T extends CmsBase, V extends FieldValues> {
  segment: CmsSegment
  row: T | null
  fields: readonly Path<V>[]
  setError: UseFormSetError<V>
  toBody: (values: V) => Record<string, unknown>
  onDone: () => void
  label: string
}

export function useCmsSave<T extends CmsBase, V extends FieldValues>({ segment, row, fields, setError, toBody, onDone, label }: SaveOptions<T, V>) {
  const qc = useQueryClient()
  const toast = useToast()
  const [formError, setFormError] = useState<string | null>(null)
  const mutation = useMutation({
    mutationFn: (values: V) => {
      const svc = cmsApi<T>(segment)
      return row ? svc.update(row.id, toBody(values)) : svc.create(toBody(values))
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: cmsKey(segment) })
      toast.success(`${label} saved.`)
      onDone()
    },
    onError: (err) => setFormError(applyServerErrors(err, setError, fields)),
  })
  return { mutation, formError }
}

interface FormFrameProps {
  onSubmit: SubmitEventHandler<HTMLFormElement>
  onCancel: () => void
  pending: boolean
  formError: string | null
  children: ReactNode
}

export function CmsFormFrame({ onSubmit, onCancel, pending, formError, children }: FormFrameProps) {
  return (
    <form onSubmit={onSubmit} noValidate className="space-y-4">
      {formError ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
          {formError}
        </p>
      ) : null}
      <div className="space-y-4">{children}</div>
      <div className="flex justify-end gap-2 border-t border-slate-200 pt-4">
        <Button variant="secondary" onClick={onCancel} disabled={pending}>
          Cancel
        </Button>
        <Button type="submit" loading={pending}>
          Save
        </Button>
      </div>
    </form>
  )
}

export function nullable(value: string): string | null {
  return value.trim() === '' ? null : value.trim()
}
