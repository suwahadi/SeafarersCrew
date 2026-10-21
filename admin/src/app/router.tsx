import { lazy } from 'react'
import { createBrowserRouter, Link } from 'react-router'
import { AppLayout } from './AppLayout'
import { RequireAuth } from './auth'
import { moduleRoutes } from './modules'

const LoginPage = lazy(() => import('@/pages/auth/LoginPage'))
const RecoveryPage = lazy(() => import('@/pages/auth/RecoveryPage'))
const ChangePasswordPage = lazy(() => import('@/pages/auth/ChangePasswordPage'))

function NotFound() {
  return (
    <div className="py-16 text-center">
      <h1 className="text-xl font-semibold text-navy-800">Page not found</h1>
      <p className="mt-2 text-sm text-slate-600">The page you requested does not exist.</p>
      <Link to="/" className="mt-4 inline-block text-sm text-navy-700 hover:underline">
        Back to the dashboard
      </Link>
    </div>
  )
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/recovery', element: <RecoveryPage /> },
  {
    path: '/change-password',
    element: (
      <RequireAuth>
        <ChangePasswordPage />
      </RequireAuth>
    ),
  },
  {
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [...moduleRoutes, { path: '*', element: <NotFound /> }],
  },
])
