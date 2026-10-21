import { lazy } from 'react'
import type { RouteObject } from 'react-router'
import { OptionsReady } from '@/components/forms/OptionsReady'

const ContractListPage = lazy(() => import('./ContractListPage'))
const ContractFormPage = lazy(() => import('./ContractFormPage'))

export const contractRoutes: RouteObject[] = [
  { path: 'contracts', element: <ContractListPage /> },
  { path: 'contracts/new', element: <OptionsReady><ContractFormPage /></OptionsReady> },
  { path: 'contracts/:id', element: <OptionsReady><ContractFormPage /></OptionsReady> },
]
