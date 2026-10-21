import { lazy } from 'react'
import type { RouteObject } from 'react-router'
import { OptionsReady } from '@/components/forms/OptionsReady'

const CrewListPage = lazy(() => import('./CrewListPage'))
const CrewCreatePage = lazy(() => import('./CrewCreatePage'))
const CrewImportPage = lazy(() => import('./CrewImportPage'))
const CrewDetailPage = lazy(() => import('./CrewDetailPage'))

export const crewRoutes: RouteObject[] = [
  { path: 'crews', element: <CrewListPage /> },
  { path: 'crews/new', element: <OptionsReady><CrewCreatePage /></OptionsReady> },
  { path: 'crews/import', element: <CrewImportPage /> },
  { path: 'crews/:id', element: <OptionsReady><CrewDetailPage /></OptionsReady> },
]
