import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { applicantsApi, type Stage } from '@/api/applicants'
import { errorMessage } from '@/api/client'
import { useToast } from '@/components/ui/Toast'
import { STAGE_LABELS } from '@/lib/pipeline'

export interface MoveRequest {
  id: string
  name: string
  from: Stage
  to: Stage
}

export function useStageMove(onDone?: () => void) {
  const qc = useQueryClient()
  const toast = useToast()
  const [request, setRequest] = useState<MoveRequest | null>(null)
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: ({ req, note }: { req: MoveRequest; note: string }) => applicantsApi.setStage(req.id, req.to, note),
    onSuccess: (detail, { req }) => {
      qc.setQueryData(['applicant', req.id], detail)
      void qc.invalidateQueries({ queryKey: ['applicants'] })
      void qc.invalidateQueries({ queryKey: ['grid', 'applicants'] })
      void qc.invalidateQueries({ queryKey: ['grid', 'vacancies'] })
      toast.success(`${req.name} moved to ${STAGE_LABELS[req.to]}.`)
      setRequest(null)
      setError(null)
      onDone?.()
    },
    onError: (err) => {
      setError(errorMessage(err))
      void qc.invalidateQueries({ queryKey: ['applicants'] })
    },
  })

  return {
    request,
    error,
    loading: mutation.isPending,
    start: (req: MoveRequest) => {
      setError(null)
      setRequest(req)
    },
    cancel: () => {
      setRequest(null)
      setError(null)
    },
    confirm: (note: string) => {
      if (request) mutation.mutate({ req: request, note })
    },
  }
}
