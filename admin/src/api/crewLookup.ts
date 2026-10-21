import { api } from './client'

export interface CrewLookupRow {
  id: string
  full_name: string
  rank_name: string
  nationality: string
  status: { code: string; name: string; color: string }
}

export const crewLookupApi = {
  search: async (q: string, signal?: AbortSignal): Promise<CrewLookupRow[]> => {
    const res = await api.list<{ items: CrewLookupRow[] }>('/crews', { q, per_page: 10 }, signal)
    return res.data.items
  },
}
