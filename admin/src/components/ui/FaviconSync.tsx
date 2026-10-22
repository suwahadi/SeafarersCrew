import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { BRANDING_KEY, brandingApi } from '@/api/branding'
import { applyFavicon, faviconLink } from '@/lib/favicon'

export function FaviconSync() {
  const branding = useQuery({ queryKey: BRANDING_KEY, queryFn: ({ signal }) => brandingApi.get(signal), staleTime: 5 * 60_000, retry: false })
  const { href, type } = faviconLink(branding.data?.favicon)
  useEffect(() => {
    applyFavicon(document, { href, type })
  }, [href, type])
  return null
}
