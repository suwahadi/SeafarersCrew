import type { AdminImage } from '@/api/cmsMedia'

interface ThumbProps {
  image: AdminImage | null | undefined
  width: number
  height: number
  alt?: string
  fit?: 'cover' | 'contain'
  fallback?: string
  fluid?: boolean
}

export function Thumb({ image, width, height, alt = '', fit = 'contain', fallback, fluid = false }: ThumbProps) {
  return (
    <div
      className={`flex shrink-0 items-center justify-center overflow-hidden rounded bg-slate-100 ring-1 ring-inset ring-slate-200 ${fluid ? 'w-full' : ''}`}
      style={fluid ? { aspectRatio: `${width} / ${height}` } : { width, height }}
    >
      {image ? (
        <img src={image.thumb_url || image.url} alt={alt} width={width} height={height} loading="lazy" className={`h-full w-full ${fit === 'cover' ? 'object-cover' : 'object-contain'}`} />
      ) : fallback ? (
        <span className="text-xs font-semibold uppercase text-slate-500">{fallback}</span>
      ) : (
        <span className="text-xs text-slate-400">—</span>
      )}
    </div>
  )
}
