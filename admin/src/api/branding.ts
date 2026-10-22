import { api } from './client'

export interface BrandingFavicon {
  url: string
  mime_type: string
}

export interface Branding {
  favicon: BrandingFavicon | null
}

export const BRANDING_KEY = ['branding'] as const

export const brandingApi = {
  get: (signal?: AbortSignal) => api.get<Branding>('/branding', undefined, signal),
}
