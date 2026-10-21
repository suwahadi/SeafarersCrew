import type { CSSProperties } from 'react'
import { buildSrcSet } from '@/lib/images'
import type { MaybeImage } from '@/lib/types'

interface SiteImageProps {
  image: MaybeImage | undefined
  sizes: string
  priority?: boolean
  eager?: boolean
  className?: string
  alt?: string
  style?: CSSProperties
}

export function SiteImage({ image, sizes, priority = false, eager = false, className, alt, style }: SiteImageProps) {
  if (!image) {
    return null
  }
  const srcSet = buildSrcSet(image)
  return (
    <img
      src={image.url}
      srcSet={srcSet}
      sizes={srcSet ? sizes : undefined}
      width={image.width > 0 ? image.width : undefined}
      height={image.height > 0 ? image.height : undefined}
      alt={alt ?? image.alt}
      loading={priority || eager ? 'eager' : 'lazy'}
      decoding="async"
      fetchPriority={priority ? 'high' : undefined}
      className={className}
      style={style}
    />
  )
}
