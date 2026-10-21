import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const HeroSlidesPage = lazy(() => import('./HeroSlidesPage'))
const GalleryPage = lazy(() => import('./GalleryPage'))
const TestimonialsPage = lazy(() => import('./TestimonialsPage'))
const ClientLogosPage = lazy(() => import('./ClientLogosPage'))
const StatsPage = lazy(() => import('./StatsPage'))
const LanguagesPage = lazy(() => import('./LanguagesPage'))

export const cmsListRoutes: RouteObject[] = [
  { path: 'website/hero-slides', element: <HeroSlidesPage /> },
  { path: 'website/gallery', element: <GalleryPage /> },
  { path: 'website/testimonials', element: <TestimonialsPage /> },
  { path: 'website/client-logos', element: <ClientLogosPage /> },
  { path: 'website/stats', element: <StatsPage /> },
  { path: 'website/languages', element: <LanguagesPage /> },
]
