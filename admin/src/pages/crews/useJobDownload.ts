import { useMutation } from '@tanstack/react-query'
import type { Job } from '@/api/client'
import { filesApi } from '@/api/files'
import { waitForJob } from '@/api/jobs'

interface FileResult {
  file_id?: string
}

export function useJobDownload(start: () => Promise<Job>) {
  return useMutation({
    mutationFn: async () => {
      const queued = await start()
      const job = await waitForJob(queued.id)
      if (job.status === 'failed') throw new Error(job.error ?? 'The export failed.')
      const fileId = (job.result as FileResult | undefined)?.file_id
      if (!fileId) throw new Error('The export finished without a file.')
      const { url } = await filesApi.url(fileId)
      window.location.assign(url)
      return job
    },
  })
}

export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
