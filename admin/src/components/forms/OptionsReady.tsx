import type { ReactNode } from 'react'
import { errorMessage } from '@/api/client'
import { Spinner } from '@/components/ui/Spinner'
import { useOptions } from '@/hooks/useOptions'

export function OptionsReady({ children }: { children: ReactNode }) {
  const { isPending, error } = useOptions()
  if (error) return <p role="alert" className="text-sm text-red-700">{errorMessage(error)}</p>
  if (isPending) {
    return (
      <div className="flex justify-center p-12 text-navy-800">
        <Spinner size="lg" label="Loading master data" />
      </div>
    )
  }
  return children
}
