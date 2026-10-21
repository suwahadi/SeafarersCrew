import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const PageListPage = lazy(() => import('./PageListPage'))
const PageFormPage = lazy(() => import('./PageFormPage'))

export const pageRoutes: RouteObject[] = [
  { path: 'website/pages', element: <PageListPage /> },
  { path: 'website/pages/new', element: <PageFormPage /> },
  { path: 'website/pages/:id', element: <PageFormPage /> },
]
