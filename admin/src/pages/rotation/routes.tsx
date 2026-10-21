import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const RotationPage = lazy(() => import('./RotationPage'))

export const rotationRoutes: RouteObject[] = [{ path: 'rotation', element: <RotationPage /> }]
