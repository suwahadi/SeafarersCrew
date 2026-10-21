import type { Metadata } from 'next'
import { pickSourceNear } from '@/lib/images'
import type { MaybeImage, Site } from '@/lib/types'

const OG_TARGET_WIDTH = 1200
const DEFAULT_PATTERN = '%s'

export function titleTemplate(pattern: string): string {
  return pattern.includes('%s') ? pattern : DEFAULT_PATTERN
}

export function applyTitlePattern(pattern: string, title: string): string {
  const effective = pattern.includes('%s') ? pattern : DEFAULT_PATTERN
  return effective.replace('%s', () => title)
}

export function pageTitle(site: Pick<Site, 'seo'>, title: string): string {
  return applyTitlePattern(site.seo.meta_title_pattern, title)
}

export function homeTitle(site: Pick<Site, 'name' | 'tagline'>, seoTitle?: string): string {
  if (seoTitle) {
    return seoTitle
  }
  return site.tagline ? `${site.name} | ${site.tagline}` : site.name
}

export function descriptionOrDefault(site: Pick<Site, 'seo'>, description?: string): string {
  return description?.trim() ? description.trim() : site.seo.meta_description
}

export function normalizePath(path: string): string {
  const withSlash = path.startsWith('/') ? path : `/${path}`
  return withSlash.length > 1 ? withSlash.replace(/\/+$/, '') : withSlash
}

export function canonicalUrl(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/\/+$/, '')}${normalizePath(path)}`
}

export interface OgImage {
  url: string
  width: number
  height: number
  alt: string
}

export function resolveOgImage(preferred: MaybeImage | undefined, fallback: MaybeImage): OgImage | null {
  const image = preferred ?? fallback
  if (!image) {
    return null
  }
  const source = pickSourceNear(image, OG_TARGET_WIDTH)
  return { url: source.url, width: source.width, height: source.height, alt: image.alt }
}

export interface MetadataInput {
  site: Site
  baseUrl: string
  path: string
  title: string
  description?: string
  image?: MaybeImage
  type?: 'website' | 'article'
  publishedTime?: string
  modifiedTime?: string
}

export function buildMetadata(input: MetadataInput): Metadata {
  const { site, baseUrl, path, title } = input
  const description = descriptionOrDefault(site, input.description)
  const url = canonicalUrl(baseUrl, path)
  const og = resolveOgImage(input.image, site.seo.default_og_image)
  const images = og ? [{ url: og.url, width: og.width, height: og.height, alt: og.alt }] : undefined
  const common = { title, description, url, siteName: site.name, images }

  return {
    title: { absolute: title },
    description,
    alternates: { canonical: url },
    openGraph:
      input.type === 'article'
        ? { ...common, type: 'article', publishedTime: input.publishedTime, modifiedTime: input.modifiedTime }
        : { ...common, type: 'website' },
    twitter: { card: 'summary_large_image', title, description, images: og ? [og.url] : undefined },
  }
}
