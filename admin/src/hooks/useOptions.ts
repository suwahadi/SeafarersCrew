import { useQuery } from '@tanstack/react-query'
import { OPTIONS_KEY, optionsApi, type MasterOptions } from '@/api/options'

const EMPTY: MasterOptions = {
  companies: [],
  vessels: [],
  ranks: [],
  vessel_types: [],
  currencies: [],
  crew_statuses: [],
  allowance_types: [],
  document_types: [],
  certificate_types: [],
}

export function useOptions() {
  const query = useQuery({ queryKey: OPTIONS_KEY, queryFn: ({ signal }) => optionsApi.all(signal), staleTime: 5 * 60_000 })
  return { options: query.data ?? EMPTY, isPending: query.isPending, error: query.error }
}
