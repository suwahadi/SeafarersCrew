import { SiteImage } from '@/components/ui/site-image'
import type { Site } from '@/lib/types'

export function Maintenance({ site }: { site: Site }) {
  const logo = site.branding.logo_light ?? site.branding.logo_dark
  return (
    <main id="main" className="flex min-h-dvh flex-col items-center justify-center gap-6 px-6 text-center">
      {logo ? (
        <SiteImage image={logo} sizes="200px" eager alt={site.name} className="h-14 w-auto max-w-56 object-contain" />
      ) : (
        <p translate="no" className="text-2xl font-bold text-primary">
          {site.name}
        </p>
      )}
      <h1 className="text-3xl font-bold text-primary">We will be back soon</h1>
      <p className="max-w-md text-muted">
        The website is undergoing scheduled maintenance. Please check back in a little while.
      </p>
    </main>
  )
}
