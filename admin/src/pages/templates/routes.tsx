import { lazy } from 'react'
import type { RouteObject } from 'react-router'

const EmailTemplatesPage = lazy(() => import('./EmailTemplatesPage'))
const EmailTemplateEditPage = lazy(() => import('./EmailTemplateEditPage'))
const DocumentTemplatesPage = lazy(() => import('./DocumentTemplatesPage'))
const DocumentTemplateEditPage = lazy(() => import('./DocumentTemplateEditPage'))

export const templateRoutes: RouteObject[] = [
  { path: 'email-templates', element: <EmailTemplatesPage /> },
  { path: 'email-templates/:code', element: <EmailTemplateEditPage /> },
  { path: 'document-templates', element: <DocumentTemplatesPage /> },
  { path: 'document-templates/new', element: <DocumentTemplateEditPage /> },
  { path: 'document-templates/:id', element: <DocumentTemplateEditPage /> },
]
