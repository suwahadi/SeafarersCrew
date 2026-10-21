import type { ReactNode } from 'react'
import { Breadcrumbs } from '@/components/ui/breadcrumbs'
import { Container } from '@/components/ui/container'
import { JsonLd } from '@/components/ui/json-ld'
import { RichHtml } from '@/components/ui/rich-html'
import { appUrl } from '@/lib/env'
import { breadcrumbJsonLd, type Crumb } from '@/lib/jsonld'

interface PageHeaderProps {
  title: string
  crumbs: readonly Crumb[]
  introHtml?: string
  children?: ReactNode
}

export function PageHeader({ title, crumbs, introHtml = '', children }: PageHeaderProps) {
  return (
    <header className="bg-primary text-on-primary">
      <Container className="py-10 sm:py-14">
        <Breadcrumbs crumbs={crumbs} />
        <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">{title}</h1>
        <RichHtml html={introHtml} className="prose-invert mt-4 max-w-3xl text-lg opacity-90" />
        {children}
      </Container>
      <JsonLd node={breadcrumbJsonLd(crumbs, appUrl())} />
    </header>
  )
}
