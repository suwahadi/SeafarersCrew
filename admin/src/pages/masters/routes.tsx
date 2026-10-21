import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const MasterPage = lazy(() => import('./MasterPage'))
const ComplianceMatrixPage = lazy(() => import('./ComplianceMatrixPage'))

export const masterRoutes: RouteObject[] = [
  { path: 'masters/:master', element: <MasterPage /> },
  { path: 'compliance-matrix', element: <ComplianceMatrixPage /> },
]
