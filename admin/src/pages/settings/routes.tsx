import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const SettingsPage = lazy(() => import('./SettingsPage'))
const AccountPage = lazy(() => import('./AccountPage'))

export const settingsRoutes: RouteObject[] = [
  { path: 'settings', element: <SettingsPage /> },
  { path: 'account', element: <AccountPage /> },
]
