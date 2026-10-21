import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const MenusPage = lazy(() => import('./MenusPage'))

export const menuRoutes: RouteObject[] = [{ path: 'website/menus', element: <MenusPage /> }]
