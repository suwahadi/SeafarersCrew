import { useQuery, useQueryClient } from '@tanstack/react-query'
import { createContext, useContext, useEffect, type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router'
import { authApi, type Admin, type Session } from '@/api/auth'
import { ApiError, setUnauthenticatedHandler } from '@/api/client'
import { Spinner } from '@/components/ui/Spinner'

export const SESSION_KEY = ['session'] as const

interface AuthState {
  admin: Admin | null
  loading: boolean
}

const AuthContext = createContext<AuthState>({ admin: null, loading: true })

export function AuthProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient()
  const session = useQuery<Session | null>({
    queryKey: SESSION_KEY,
    queryFn: async () => {
      try {
        return await authApi.me()
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) return null
        throw err
      }
    },
    staleTime: 5 * 60_000,
    retry: false,
  })

  useEffect(() => {
    setUnauthenticatedHandler(() => {
      qc.setQueryData(SESSION_KEY, null)
    })
  }, [qc])

  return <AuthContext.Provider value={{ admin: session.data?.admin ?? null, loading: session.isPending }}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  return useContext(AuthContext)
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { admin, loading } = useAuth()
  const location = useLocation()
  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center text-navy-800">
        <Spinner size="lg" label="Checking session" />
      </div>
    )
  }
  if (!admin) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  if (admin.must_change_password && location.pathname !== '/change-password') return <Navigate to="/change-password" replace />
  return <>{children}</>
}
