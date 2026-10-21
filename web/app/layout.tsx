import './globals.css'
import type { Metadata, Viewport } from 'next'
import localFont from 'next/font/local'
import { cookies } from 'next/headers'
import { connection } from 'next/server'
import { Suspense, type CSSProperties, type ReactNode } from 'react'
import { preconnect } from 'react-dom'
import { Footer } from '@/components/layout/footer'
import { Header } from '@/components/layout/header'
import { Maintenance } from '@/components/layout/maintenance'
import { NavigationSpinner } from '@/components/layout/navigation-spinner'
import { WhatsAppButton } from '@/components/layout/whatsapp-button'
import { JsonLd } from '@/components/ui/json-ld'
import { getSite } from '@/lib/api'
import { brandTheme } from '@/lib/contrast'
import { appUrl } from '@/lib/env'
import { organizationJsonLd } from '@/lib/jsonld'
import { list } from '@/lib/list'
import { TRANSLATE_COOKIE, targetFromCookieValue } from '@/lib/language'
import { homeTitle, titleTemplate } from '@/lib/seo'
import type { Site } from '@/lib/types'

const inter = localFont({
  src: '../node_modules/@fontsource-variable/inter/files/inter-latin-wght-normal.woff2',
  variable: '--font-inter',
  display: 'swap',
  weight: '100 900',
  fallback: ['ui-sans-serif', 'system-ui', 'Segoe UI', 'Roboto', 'Arial', 'sans-serif'],
})

export async function generateMetadata(): Promise<Metadata> {
  await connection()
  const site = await getSite()
  const favicon = site.branding.favicon
  return {
    metadataBase: new URL(appUrl()),
    title: { default: homeTitle(site), template: titleTemplate(site.seo.meta_title_pattern) },
    description: site.seo.meta_description,
    applicationName: site.name,
    icons: favicon ? { icon: favicon.url, shortcut: favicon.url, apple: favicon.url } : { icon: '/favicon.svg' },
    verification: site.seo.google_site_verification ? { google: site.seo.google_site_verification } : undefined,
    robots: site.seo.robots_indexing ? { index: true, follow: true } : { index: false, follow: false },
  }
}

export async function generateViewport(): Promise<Viewport> {
  await connection()
  const site = await getSite()
  return { themeColor: brandTheme(site.branding.primary_color, site.branding.accent_color).primary, width: 'device-width', initialScale: 1 }
}

function sourceLanguageCode(site: Site): string {
  return list(site.languages).find((language) => language.is_source)?.code ?? site.default_language
}

function themeStyle(site: Site): CSSProperties {
  const theme = brandTheme(site.branding.primary_color, site.branding.accent_color)
  return {
    '--color-primary': theme.primary,
    '--color-on-primary': theme.onPrimary,
    '--color-accent': theme.accent,
    '--color-on-accent': theme.onAccent,
    '--color-accent-text': theme.accentText,
  } as CSSProperties
}

function preconnectToMedia(site: Site): void {
  const sample = site.branding.logo_light ?? site.branding.logo_dark ?? site.seo.default_og_image
  if (sample) {
    preconnect(new URL(sample.url).origin)
  }
}

export default async function RootLayout({ children }: { children: ReactNode }) {
  await connection()
  const [site, cookieStore] = await Promise.all([getSite(), cookies()])
  const source = sourceLanguageCode(site)
  const activeLanguage = targetFromCookieValue(cookieStore.get(TRANSLATE_COOKIE)?.value, source) ?? source
  preconnectToMedia(site)

  return (
    <html lang={site.default_language || 'en'} className={inter.variable} style={themeStyle(site)} suppressHydrationWarning>
      <body className="flex min-h-dvh flex-col">
        {site.maintenance_mode ? (
          <Maintenance site={site} />
        ) : (
          <>
            <a href="#main" className="skip-link">
              Skip to main content
            </a>
            <Header site={site} activeLanguage={activeLanguage} />
            <main id="main" className="flex-1">
              {children}
            </main>
            <Footer site={site} />
            <WhatsAppButton whatsapp={site.contact.whatsapp} />
            <Suspense fallback={null}>
              <NavigationSpinner />
            </Suspense>
          </>
        )}
        <JsonLd node={organizationJsonLd(site, appUrl())} />
      </body>
    </html>
  )
}
