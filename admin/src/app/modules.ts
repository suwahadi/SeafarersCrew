import type { RouteObject } from 'react-router'
import { crewRoutes } from '@/pages/crews/routes'
import { dashboardRoutes } from '@/pages/dashboard/routes'

export const moduleRoutes: RouteObject[] = [
  ...dashboardRoutes,
  ...crewRoutes,
]
