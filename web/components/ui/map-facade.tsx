'use client'

import { useState } from 'react'
import { Icon } from '@/components/ui/icons'
import { mapEmbedUrl, mapViewUrl, type Coordinates } from '@/lib/map'

interface MapFacadeProps {
  coordinates: Coordinates
  address: string
}

export function MapFacade({ coordinates, address }: MapFacadeProps) {
  const [open, setOpen] = useState(false)
  return (
    <div>
      <div className="relative aspect-[4/3] w-full overflow-hidden rounded-lg border border-line bg-surface sm:aspect-[16/9]">
        {open ? (
          <iframe
            title="Office location map"
            src={mapEmbedUrl(coordinates)}
            loading="lazy"
            referrerPolicy="no-referrer"
            className="absolute inset-0 size-full border-0"
          />
        ) : (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 p-6 text-center">
            <Icon name="map-pin" className="size-10 text-accent-text" />
            {address ? <p className="max-w-sm font-medium text-ink">{address}</p> : null}
            <button
              type="button"
              onClick={() => {
                setOpen(true)
              }}
              className="rounded-md bg-primary px-5 py-2.5 text-sm font-semibold text-on-primary transition-colors duration-200 ease-out hover:bg-primary-hover"
            >
              Open map
            </button>
          </div>
        )}
      </div>
      <p className="mt-2 text-sm">
        <a
          href={mapViewUrl(coordinates)}
          target="_blank"
          rel="noopener noreferrer"
          className="font-semibold text-accent-text underline underline-offset-2"
        >
          View larger map
        </a>
      </p>
    </div>
  )
}
