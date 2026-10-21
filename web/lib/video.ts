import type { VideoSettings } from '@/lib/types'

const SAFE_VIDEO_ID = /^[A-Za-z0-9_-]{3,32}$/

export function embedSrc(video: Pick<VideoSettings, 'provider' | 'video_id'>): string {
  if (!SAFE_VIDEO_ID.test(video.video_id)) {
    return ''
  }
  const id = encodeURIComponent(video.video_id)
  switch (video.provider) {
    case 'youtube':
      return `https://www.youtube-nocookie.com/embed/${id}?autoplay=1&rel=0`
    case 'vimeo':
      return `https://player.vimeo.com/video/${id}?autoplay=1&dnt=1`
    default:
      return ''
  }
}
