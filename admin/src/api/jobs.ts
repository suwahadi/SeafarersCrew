import { api, type Job } from './client'

export const jobsApi = {
  get: (id: string) => api.get<Job>(`/jobs/${id}`),
}

export async function waitForJob(id: string, opts: { intervalMs?: number; timeoutMs?: number } = {}): Promise<Job> {
  const interval = opts.intervalMs ?? 800
  const deadline = Date.now() + (opts.timeoutMs ?? 5 * 60_000)
  for (;;) {
    const job = await jobsApi.get(id)
    if (job.status === 'succeeded' || job.status === 'failed') return job
    if (Date.now() > deadline) throw new Error('The job is taking longer than expected; check back later.')
    await new Promise((resolve) => setTimeout(resolve, interval))
  }
}
