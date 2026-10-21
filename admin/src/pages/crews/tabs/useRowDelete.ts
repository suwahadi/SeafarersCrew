import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage } from '@/api/client'
import { useToast } from '@/components/ui/Toast'

export function useRowDelete<T>(remove: (row: T) => Promise<unknown>, invalidate: QueryKey[], label: string) {
  const qc = useQueryClient()
  const toast = useToast()
  const [target, setTarget] = useState<T | null>(null)
  const mutation = useMutation({
    mutationFn: remove,
    onSuccess: () => {
      for (const key of invalidate) void qc.invalidateQueries({ queryKey: key })
      toast.success(`${label} deleted.`)
      setTarget(null)
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
  return {
    target,
    ask: setTarget,
    cancel: () => setTarget(null),
    confirm: () => {
      if (target !== null) mutation.mutate(target)
    },
    pending: mutation.isPending,
  }
}
