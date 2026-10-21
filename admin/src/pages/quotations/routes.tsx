import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const QuotationListPage = lazy(() => import('./QuotationListPage'))
const QuotationFormPage = lazy(() => import('./QuotationFormPage'))

export const quotationRoutes: RouteObject[] = [
  { path: 'quotations', element: <QuotationListPage /> },
  { path: 'quotations/new', element: <QuotationFormPage /> },
  { path: 'quotations/:id', element: <QuotationFormPage /> },
]
