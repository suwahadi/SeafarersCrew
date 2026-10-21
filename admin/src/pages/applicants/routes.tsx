import { lazy } from 'react'
import type { RouteObject } from 'react-router'
import { OptionsReady } from '@/components/forms/OptionsReady'

const ApplicantListPage = lazy(() => import('./ApplicantListPage'))
const ApplicantDetailPage = lazy(() => import('./ApplicantDetailPage'))

export const applicantRoutes: RouteObject[] = [
  { path: 'applicants', element: <OptionsReady><ApplicantListPage /></OptionsReady> },
  { path: 'applicants/:id', element: <ApplicantDetailPage /> },
]
