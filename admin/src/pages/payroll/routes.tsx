import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const PayrollPeriodListPage = lazy(() => import('./PayrollPeriodListPage'))
const PayrollPeriodPage = lazy(() => import('./PayrollPeriodPage'))

export const payrollRoutes: RouteObject[] = [
  { path: 'payroll', element: <PayrollPeriodListPage /> },
  { path: 'payroll/:id', element: <PayrollPeriodPage /> },
]
