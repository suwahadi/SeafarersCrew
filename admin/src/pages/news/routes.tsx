import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const NewsListPage = lazy(() => import('./NewsListPage'))
const NewsFormPage = lazy(() => import('./NewsFormPage'))

export const newsRoutes: RouteObject[] = [
  { path: 'website/news', element: <NewsListPage /> },
  { path: 'website/news/new', element: <NewsFormPage /> },
  { path: 'website/news/:id', element: <NewsFormPage /> },
]
