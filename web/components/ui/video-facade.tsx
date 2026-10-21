'use client'

import { useState } from 'react'
import { Icon } from '@/components/ui/icons'
import { embedSrc } from '@/lib/video'
import type { VideoSettings } from '@/lib/types'

export function VideoFacade({ video }: { video: VideoSettings }) {
  const [playing, setPlaying] = useState(false)
  const src = embedSrc(video)
  const title = video.title || 'Video'

  if (src === '') {
    return null
  }

  return (
    <div className="relative aspect-video w-full overflow-hidden rounded-lg bg-primary">
      {playing ? (
        <iframe
          src={src}
          title={title}
          className="absolute inset-0 size-full"
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; fullscreen"
          allowFullScreen
          referrerPolicy="strict-origin-when-cross-origin"
        />
      ) : (
        <button
          type="button"
          onClick={() => {
            setPlaying(true)
          }}
          aria-label={`Play video: ${title}`}
          className="group absolute inset-0 flex size-full items-center justify-center"
        >
          {video.thumbnail_url ? (
            <img
              src={video.thumbnail_url}
              alt=""
              width={1280}
              height={720}
              loading="lazy"
              decoding="async"
              referrerPolicy="no-referrer"
              className="absolute inset-0 size-full object-cover"
            />
          ) : null}
          <span className="relative inline-flex size-16 items-center justify-center rounded-full bg-white/95 text-primary shadow-lg transition-transform duration-200 ease-out group-hover:scale-110">
            <Icon name="play" className="ml-1 size-7" />
          </span>
        </button>
      )}
    </div>
  )
}
