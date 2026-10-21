import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const DocumentListPage = lazy(() => import('./DocumentListPage'))
const DocumentNewPage = lazy(() => import('./DocumentNewPage'))

export const documentRoutes: RouteObject[] = [
  { path: 'documents', element: <DocumentListPage /> },
  { path: 'documents/new', element: <DocumentNewPage /> },
]
