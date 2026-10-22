import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Suspense, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router'
import { authApi } from '@/api/auth'
import { contactMessagesApi, UNREAD_COUNT_KEY } from '@/api/contactMessages'
import { Spinner } from '@/components/ui/Spinner'
import { SESSION_KEY, useAuth } from './auth'
import { navigation, type NavItem } from './navigation'

export function AppLayout() {
  const { admin } = useAuth()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)
  const unread = useQuery({
    queryKey: UNREAD_COUNT_KEY,
    queryFn: ({ signal }) => contactMessagesApi.unreadCount(signal),
    staleTime: 60_000,
    refetchInterval: 120_000,
  })
  const badgeCount = (item: NavItem): number => (item.badge === 'unread-contact-messages' ? (unread.data?.unread ?? 0) : 0)
  const logout = useMutation({
    mutationFn: authApi.logout,
    onSettled: () => {
      qc.clear()
      qc.setQueryData(SESSION_KEY, null)
      void navigate('/login', { replace: true })
    },
  })

  return (
    <div className="min-h-screen lg:flex">
      <a href="#main" className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:bg-white focus:p-2">
        Skip to content
      </a>
      <aside
        className={`${menuOpen ? 'block' : 'hidden'} w-full shrink-0 bg-navy-800 text-navy-100 [scrollbar-color:var(--color-navy-600)_transparent] lg:sticky lg:top-0 lg:block lg:h-screen lg:w-64 lg:overflow-y-auto`}
      >
        <div className="border-b border-navy-700 px-5 py-4">
          <span className="text-lg font-semibold tracking-tight text-white">Seafarers Crew</span>
        </div>
        <nav aria-label="Main" className="space-y-5 px-3 pt-4 pb-6">
          {navigation.map((group) => (
            <div key={group.label}>
              <p className="px-2 text-xs font-semibold tracking-wide text-navy-200 uppercase">{group.label}</p>
              <ul className="mt-1 space-y-0.5">
                {group.items.map((item) => (
                  <li key={item.to}>
                    <NavLink
                      to={item.to}
                      end={item.to === '/'}
                      onClick={() => setMenuOpen(false)}
                      className={({ isActive }) =>
                        `block rounded-md px-2 py-1.5 text-sm transition-colors duration-150 ${isActive ? 'bg-navy-700 text-white' : 'hover:bg-navy-700/60 hover:text-white'}`
                      }
                    >
                      {item.label}
                      {badgeCount(item) > 0 ? (
                        <span className="ml-2 inline-flex min-w-5 items-center justify-center rounded-full bg-white px-1.5 text-xs font-semibold text-navy-800">
                          {badgeCount(item)}
                          <span className="sr-only"> unread</span>
                        </span>
                      ) : null}
                    </NavLink>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>
      </aside>
      <div className="min-w-0 flex-1">
        <header className="sticky top-0 z-10 flex items-center justify-between gap-4 border-b border-slate-200 bg-white px-4 py-2 lg:px-8">
          <button type="button" className="h-10 rounded-md px-3 text-sm ring-1 ring-slate-300 lg:hidden" aria-expanded={menuOpen} onClick={() => setMenuOpen((o) => !o)}>
            Menu
          </button>
          <div className="ml-auto flex items-center gap-3 text-sm">
            <NavLink to="/account" className="text-slate-700 hover:text-navy-800">
              {admin?.name}
            </NavLink>
            <button type="button" onClick={() => logout.mutate()} disabled={logout.isPending} className="rounded-md px-2 py-1 text-slate-600 hover:bg-slate-100">
              Sign out
            </button>
          </div>
        </header>
        <main id="main" className="mx-auto max-w-[1600px] px-4 py-6 lg:px-8">
          <Suspense
            fallback={
              <div className="flex justify-center p-12 text-navy-800">
                <Spinner size="lg" />
              </div>
            }
          >
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  )
}
