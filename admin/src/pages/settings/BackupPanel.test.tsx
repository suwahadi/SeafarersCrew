import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Backup } from '@/api/backups'
import { ToastProvider } from '@/components/ui/Toast'
import { BackupPanel, hasRunningBackup } from './BackupPanel'

const list = vi.fn()
const run = vi.fn()

vi.mock('@/api/backups', () => ({
  BACKUPS_KEY: ['backups'],
  backupsApi: { list: () => list() as unknown, run: () => run() as unknown },
}))

const done: Backup = { id: '1', status: 'succeeded', file_name: 'seafarers-20261021-020000.dump', size_bytes: 2048, started_at: '2026-10-21T02:00:00Z', finished_at: '2026-10-21T02:01:00Z', error: '' }
const failed: Backup = { id: '2', status: 'failed', file_name: '', size_bytes: 0, started_at: '2026-10-20T02:00:00Z', finished_at: '2026-10-20T02:00:05Z', error: 'pg_dump: connection refused' }

function renderPanel() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <ToastProvider>
        <BackupPanel />
      </ToastProvider>
    </QueryClientProvider>,
  )
}

describe('BackupPanel', () => {
  beforeEach(() => {
    list.mockReset()
    run.mockReset()
  })

  it('lists backups with their file, size and failure reason', async () => {
    list.mockResolvedValue({ data: [done, failed], meta: { page: 1, per_page: 20, total: 2 } })
    renderPanel()
    expect(await screen.findByText('seafarers-20261021-020000.dump')).toBeInTheDocument()
    expect(screen.getByText('2.0 KB')).toBeInTheDocument()
    expect(screen.getByText('pg_dump: connection refused')).toBeInTheDocument()
    expect(screen.getByText('failed')).toBeInTheDocument()
  })

  it('starts a manual backup', async () => {
    list.mockResolvedValue({ data: [done], meta: { page: 1, per_page: 20, total: 1 } })
    run.mockResolvedValue({ id: 'job', kind: 'backup.run', status: 'queued' })
    renderPanel()
    await userEvent.click(await screen.findByRole('button', { name: 'Run backup now' }))
    expect(run).toHaveBeenCalledTimes(1)
    expect(await screen.findByText('Backup started.')).toBeInTheDocument()
  })

  it('disables the button while a backup is running', async () => {
    list.mockResolvedValue({ data: [{ ...done, status: 'running', finished_at: null }], meta: { page: 1, per_page: 20, total: 1 } })
    renderPanel()
    await screen.findByText('running')
    expect(screen.getByRole('button', { name: /Run backup now/ })).toBeDisabled()
  })

  it('tells running backups apart for polling', () => {
    expect(hasRunningBackup([done, failed])).toBe(false)
    expect(hasRunningBackup([done, { ...done, status: 'running' }])).toBe(true)
  })
})
