import { api } from './client'

export type UploadContext = 'photo' | 'cms_image' | 'logo' | 'flag' | 'crew_document' | 'crew_cv' | 'applicant' | 'csv_import'

export interface StoredFile {
  id: string
  original_name: string
  mime_type: string
  size_original: number
  size_stored: number
  status: string
  is_public: boolean
  created_at: string
}

export interface PresignedUrl {
  url: string
  expires_at: string
}

export const filesApi = {
  upload: (file: File, context: UploadContext, owner?: { type: string; id: string }) => {
    const form = new FormData()
    form.set('file', file)
    form.set('context', context)
    if (owner) {
      form.set('owner_type', owner.type)
      form.set('owner_id', owner.id)
    }
    return api.upload<StoredFile>('/files', form)
  },
  url: (id: string) => api.get<PresignedUrl>(`/files/${id}/url`),
  remove: (id: string) => api.delete(`/files/${id}`),
}

export async function openFile(id: string): Promise<void> {
  const { url } = await filesApi.url(id)
  window.open(url, '_blank', 'noopener,noreferrer')
}
