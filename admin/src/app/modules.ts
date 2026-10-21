import type { RouteObject } from 'react-router'
import { crewRoutes } from '@/pages/crews/routes'
import { dashboardRoutes } from '@/pages/dashboard/routes'
import { masterRoutes } from '@/pages/masters/routes'
import { settingsRoutes } from '@/pages/settings/routes'
import { templateRoutes } from '@/pages/templates/routes'

export const moduleRoutes: RouteObject[] = [
  ...dashboardRoutes,
  ...crewRoutes,
  ...masterRoutes,
  ...settingsRoutes,
  ...templateRoutes,
]
