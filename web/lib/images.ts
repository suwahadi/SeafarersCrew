import type { ImageSource, PublicImage } from '@/lib/types'

export function buildSrcSet(image: PublicImage): string | undefined {
  const byWidth = new Map<number, ImageSource>()
  for (const source of image.sources) {
    if (source.width > 0 && !byWidth.has(source.width)) {
      byWidth.set(source.width, source)
    }
  }
  if (byWidth.size === 0) {
    return undefined
  }
  return [...byWidth.values()]
    .sort((a, b) => a.width - b.width)
    .map((source) => `${source.url} ${String(source.width)}w`)
    .join(', ')
}

export function aspectRatio(image: PublicImage): string {
  return image.width > 0 && image.height > 0 ? `${String(image.width)} / ${String(image.height)}` : '16 / 9'
}

export function pickSourceNear(image: PublicImage, targetWidth: number): { url: string; width: number; height: number } {
  const sorted = [...image.sources].sort((a, b) => a.width - b.width)
  const found = sorted.find((source) => source.width >= targetWidth) ?? sorted[sorted.length - 1]
  return found ?? { url: image.url, width: image.width, height: image.height }
}
