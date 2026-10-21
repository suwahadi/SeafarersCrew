export type VideoProvider = 'youtube' | 'vimeo'

export interface ParsedVideo {
  provider: VideoProvider
  id: string
}

const YOUTUBE_ID = /^[A-Za-z0-9_-]{11}$/
const VIMEO_ID = /^\d{1,12}$/
const YOUTUBE_HOSTS = new Set(['youtube.com', 'www.youtube.com', 'm.youtube.com', 'music.youtube.com', 'youtube-nocookie.com', 'www.youtube-nocookie.com'])
const VIMEO_HOSTS = new Set(['vimeo.com', 'www.vimeo.com', 'player.vimeo.com'])
const YOUTUBE_ID_PREFIXES = ['embed', 'shorts', 'live', 'v']

export function parseVideoUrl(raw: string): ParsedVideo | null {
  let url: URL
  try {
    url = new URL(raw.trim())
  } catch {
    return null
  }
  if (url.protocol !== 'https:' && url.protocol !== 'http:') return null
  const host = url.hostname.toLowerCase()
  const segments = url.pathname.split('/').filter((s) => s !== '')
  const first = segments[0] ?? ''

  if (host === 'youtu.be') {
    return YOUTUBE_ID.test(first) ? { provider: 'youtube', id: first } : null
  }
  if (YOUTUBE_HOSTS.has(host)) {
    const id = first === 'watch' ? (url.searchParams.get('v') ?? '') : YOUTUBE_ID_PREFIXES.includes(first) ? (segments[1] ?? '') : ''
    return YOUTUBE_ID.test(id) ? { provider: 'youtube', id } : null
  }
  if (VIMEO_HOSTS.has(host)) {
    const id = [...segments].reverse().find((s) => VIMEO_ID.test(s))
    return id === undefined ? null : { provider: 'vimeo', id }
  }
  return null
}

export function youtubeThumbnail(id: string): string {
  return `https://i.ytimg.com/vi/${id}/hqdefault.jpg`
}
