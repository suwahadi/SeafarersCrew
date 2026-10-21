import { describe, expect, it } from 'vitest'
import { parseVideoUrl, youtubeThumbnail } from './videoUrl'

describe('parseVideoUrl', () => {
  it.each([
    ['https://www.youtube.com/watch?v=dQw4w9WgXcQ', { provider: 'youtube', id: 'dQw4w9WgXcQ' }],
    ['https://youtu.be/dQw4w9WgXcQ?t=10', { provider: 'youtube', id: 'dQw4w9WgXcQ' }],
    ['https://www.youtube.com/embed/dQw4w9WgXcQ', { provider: 'youtube', id: 'dQw4w9WgXcQ' }],
    ['https://youtube.com/shorts/dQw4w9WgXcQ', { provider: 'youtube', id: 'dQw4w9WgXcQ' }],
    ['https://vimeo.com/76979871', { provider: 'vimeo', id: '76979871' }],
    ['https://player.vimeo.com/video/76979871', { provider: 'vimeo', id: '76979871' }],
    ['https://vimeo.com/channels/staffpicks/76979871', { provider: 'vimeo', id: '76979871' }],
  ])('recognises %s', (url, expected) => {
    expect(parseVideoUrl(url)).toEqual(expected)
  })

  it.each(['', 'not a url', 'https://example.com/watch?v=dQw4w9WgXcQ', 'https://www.youtube.com/watch', 'https://www.youtube.com/watch?v=short', 'javascript:alert(1)', 'https://vimeo.com/about'])(
    'rejects %j',
    (url) => {
      expect(parseVideoUrl(url)).toBeNull()
    },
  )

  it('builds the YouTube thumbnail address', () => {
    expect(youtubeThumbnail('abc')).toBe('https://i.ytimg.com/vi/abc/hqdefault.jpg')
  })
})
