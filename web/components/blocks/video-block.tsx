import { VideoFacade } from '@/components/ui/video-facade'
import type { VideoSettings } from '@/lib/types'

export function VideoBlock({ settings }: { settings: VideoSettings }) {
  return (
    <figure className="mx-auto max-w-4xl">
      <VideoFacade video={settings} />
      {settings.title ? <figcaption className="mt-3 text-center text-sm text-muted">{settings.title}</figcaption> : null}
    </figure>
  )
}
