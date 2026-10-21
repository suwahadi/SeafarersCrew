export interface AdminImage {
  file_id: string
  url: string
  thumb_url: string
  mime_type: string
  width: number
  height: number
}

export const IMAGE_ACCEPT = 'image/png,image/jpeg,image/webp'
export const LOGO_ACCEPT = 'image/png,image/jpeg,image/webp,image/svg+xml'
export const FLAG_ACCEPT = 'image/svg+xml'
