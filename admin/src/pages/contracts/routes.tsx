import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const ContractListPage = lazy(() => import('./ContractListPage'))
const ContractFormPage = lazy(() => import('./ContractFormPage'))

export const contractRoutes: RouteObject[] = [
  { path: 'contracts', element: <ContractListPage /> },
  { path: 'contracts/new', element: <ContractFormPage /> },
  { path: 'contracts/:id', element: <ContractFormPage /> },
]
