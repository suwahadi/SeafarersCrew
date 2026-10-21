import { canonicalUrl } from '@/lib/seo'
import type { NewsArticle, Site } from '@/lib/types'

export type JsonLdNode = Record<string, unknown>

export function organizationJsonLd(site: Site, baseUrl: string): JsonLdNode {
  const logo = site.branding.logo_light ?? site.branding.logo_dark
  const sameAs = Object.values(site.social).filter((link) => link !== '')
  const { organization } = site
  const node: JsonLdNode = {
    '@context': 'https://schema.org',
    '@type': 'Organization',
    name: site.name,
    legalName: organization.legal_name || site.name,
    url: baseUrl,
  }
  if (logo) {
    node.logo = logo.url
  }
  if (organization.email || site.contact.email) {
    node.email = organization.email || site.contact.email
  }
  if (organization.phone || site.contact.phones?.[0]) {
    node.telephone = organization.phone || site.contact.phones?.[0]
  }
  if (organization.address || organization.country) {
    node.address = {
      '@type': 'PostalAddress',
      streetAddress: organization.address,
      addressCountry: organization.country,
    }
  }
  if (sameAs.length > 0) {
    node.sameAs = sameAs
  }
  return node
}

export function newsArticleJsonLd(article: NewsArticle, site: Site, baseUrl: string): JsonLdNode {
  const url = canonicalUrl(baseUrl, `/news/${article.slug}`)
  const logo = site.branding.logo_light ?? site.branding.logo_dark
  const node: JsonLdNode = {
    '@context': 'https://schema.org',
    '@type': 'NewsArticle',
    headline: article.title,
    description: article.excerpt || article.meta_description,
    datePublished: article.published_at,
    dateModified: article.updated_at,
    mainEntityOfPage: url,
    url,
    author: { '@type': 'Organization', name: site.name },
    publisher: {
      '@type': 'Organization',
      name: site.name,
      ...(logo ? { logo: { '@type': 'ImageObject', url: logo.url } } : {}),
    },
  }
  if (article.thumbnail) {
    node.image = [article.thumbnail.url]
  }
  return node
}

export interface Crumb {
  name: string
  path: string
}

export function breadcrumbJsonLd(crumbs: readonly Crumb[], baseUrl: string): JsonLdNode {
  return {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: crumbs.map((crumb, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: crumb.name,
      item: canonicalUrl(baseUrl, crumb.path),
    })),
  }
}

const LINE_SEPARATOR = String.fromCharCode(0x2028)
const PARAGRAPH_SEPARATOR = String.fromCharCode(0x2029)

export function serializeJsonLd(node: JsonLdNode): string {
  return JSON.stringify(node)
    .replaceAll('<', '\\u003c')
    .replaceAll(LINE_SEPARATOR, '\\u2028')
    .replaceAll(PARAGRAPH_SEPARATOR, '\\u2029')
}
