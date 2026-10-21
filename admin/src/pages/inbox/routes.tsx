import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const ContactInboxPage = lazy(() => import('./ContactInboxPage'))

export const inboxRoutes: RouteObject[] = [{ path: 'contact-messages', element: <ContactInboxPage /> }]
