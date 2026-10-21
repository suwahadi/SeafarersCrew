import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const EmailLogPage = lazy(() => import('./EmailLogPage'))

export const emailRoutes: RouteObject[] = [{ path: 'email-logs', element: <EmailLogPage /> }]
