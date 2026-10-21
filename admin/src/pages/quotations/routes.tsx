import { lazy } from 'react'
import type { RouteObject } from 'react-router'
import { OptionsReady } from '@/components/forms/OptionsReady'

const QuotationListPage = lazy(() => import('./QuotationListPage'))
const QuotationFormPage = lazy(() => import('./QuotationFormPage'))

export const quotationRoutes: RouteObject[] = [
  { path: 'quotations', element: <QuotationListPage /> },
  { path: 'quotations/new', element: <OptionsReady><QuotationFormPage /></OptionsReady> },
  { path: 'quotations/:id', element: <OptionsReady><QuotationFormPage /></OptionsReady> },
]
