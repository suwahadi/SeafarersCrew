import type { RouteObject } from 'react-router'
import { applicantRoutes } from '@/pages/applicants/routes'
import { contractRoutes } from '@/pages/contracts/routes'
import { crewRoutes } from '@/pages/crews/routes'
import { dashboardRoutes } from '@/pages/dashboard/routes'
import { documentRoutes } from '@/pages/documents/routes'
import { emailRoutes } from '@/pages/email/routes'
import { inboxRoutes } from '@/pages/inbox/routes'
import { masterRoutes } from '@/pages/masters/routes'
import { quotationRoutes } from '@/pages/quotations/routes'
import { rotationRoutes } from '@/pages/rotation/routes'
import { settingsRoutes } from '@/pages/settings/routes'
import { templateRoutes } from '@/pages/templates/routes'
import { vacancyRoutes } from '@/pages/vacancies/routes'

export const moduleRoutes: RouteObject[] = [
  ...dashboardRoutes,
  ...crewRoutes,
  ...contractRoutes,
  ...rotationRoutes,
  ...quotationRoutes,
  ...documentRoutes,
  ...emailRoutes,
  ...vacancyRoutes,
  ...applicantRoutes,
  ...inboxRoutes,
  ...masterRoutes,
  ...settingsRoutes,
  ...templateRoutes,
]
