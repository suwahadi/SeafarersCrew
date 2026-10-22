import { mapEmbedUrl, mapViewUrl, type Coordinates } from '@/lib/map'

interface MapEmbedProps {
  coordinates: Coordinates
}

export function MapEmbed({ coordinates }: MapEmbedProps) {
  return (
    <div>
      <div className="relative aspect-[4/3] w-full overflow-hidden rounded-lg border border-line bg-surface sm:aspect-[16/9]">
        <iframe
          title="Office location map"
          src={mapEmbedUrl(coordinates)}
          loading="lazy"
          referrerPolicy="no-referrer"
          className="absolute inset-0 size-full border-0"
        />
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
