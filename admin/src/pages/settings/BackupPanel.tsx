import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/api/client'
import { BACKUPS_KEY, backupsApi, type Backup } from '@/api/backups'
import { Button } from '@/components/ui/Button'
import { Badge, Card, EmptyState } from '@/components/ui/Layout'
import { useToast } from '@/components/ui/Toast'
import { formatBytes, formatDateTime } from '@/lib/format'

const tones = { running: 'blue', succeeded: 'green', failed: 'red' } as const

export function hasRunningBackup(items: Backup[]): boolean {
  return items.some((b) => b.status === 'running')
}

export function BackupPanel() {
  const qc = useQueryClient()
  const toast = useToast()
  const history = useQuery({
    queryKey: BACKUPS_KEY,
    queryFn: ({ signal }) => backupsApi.list(signal),
    refetchInterval: (query) => (hasRunningBackup(query.state.data?.data ?? []) ? 3000 : false),
  })
  const run = useMutation({
    mutationFn: backupsApi.run,
    onSuccess: () => {
      toast.success('Backup started.')
      void qc.invalidateQueries({ queryKey: BACKUPS_KEY })
    },
    onError: (err) => toast.error(errorMessage(err)),
  })
  const items = history.data?.data ?? []
  const running = hasRunningBackup(items)

  return (
    <Card
      title="Backup history"
      actions={
        <Button loading={run.isPending || running} disabled={run.isPending || running} onClick={() => run.mutate()}>
          Run backup now
        </Button>
      }
    >
      {history.isError ? (
        <p role="alert" className="text-sm text-red-700">
          {errorMessage(history.error)}
        </p>
      ) : items.length === 0 && !history.isPending ? (
        <EmptyState title="No backups yet" />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="text-xs uppercase text-slate-500">
              <tr>
                <th className="py-2 pr-4 font-medium">Started</th>
                <th className="py-2 pr-4 font-medium">Status</th>
                <th className="py-2 pr-4 font-medium">File</th>
                <th className="py-2 pr-4 text-right font-medium">Size</th>
                <th className="py-2 font-medium">Detail</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {items.map((b) => (
                <tr key={b.id}>
                  <td className="py-2 pr-4 whitespace-nowrap">{formatDateTime(b.started_at)}</td>
                  <td className="py-2 pr-4">
                    <Badge tone={tones[b.status]}>{b.status}</Badge>
                  </td>
                  <td className="py-2 pr-4 font-mono text-xs">{b.file_name || '—'}</td>
                  <td className="py-2 pr-4 text-right tabular-nums">{b.size_bytes > 0 ? formatBytes(b.size_bytes) : '—'}</td>
                  <td className="py-2 text-slate-600">{b.error || (b.finished_at ? `Finished ${formatDateTime(b.finished_at)}` : '')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  )
}
