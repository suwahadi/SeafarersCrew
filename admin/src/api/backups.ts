import { api, type Job } from './client'

export type BackupStatus = 'running' | 'succeeded' | 'failed'

export interface Backup {
  id: string
  status: BackupStatus
  file_name: string
  size_bytes: number
  started_at: string
  finished_at: string | null
  error: string
}

export const BACKUPS_KEY = ['backups'] as const

export const backupsApi = {
  list: (signal?: AbortSignal) => api.list<Backup[]>('/backups', { per_page: 20 }, signal),
  run: () => api.post<Job>('/backups/run', {}),
}
