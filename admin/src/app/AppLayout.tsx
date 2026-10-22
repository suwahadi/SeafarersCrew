import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Suspense, useEffect, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router'
import { authApi } from '@/api/auth'
import { contactMessagesApi, UNREAD_COUNT_KEY } from '@/api/contactMessages'
import { Icon } from '@/components/ui/Icon'
import { IconButton } from '@/components/ui/IconButton'
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
  useEffect(() => {
    if (!menuOpen) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setMenuOpen(false)
    }
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', onKey)
    return () => {
      document.body.style.overflow = previous
      document.removeEventListener('keydown', onKey)
    }
  }, [menuOpen])
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
      <div
        aria-hidden="true"
        onClick={() => setMenuOpen(false)}
        className={`fixed inset-0 z-30 bg-slate-900/50 transition-opacity duration-200 ease-out lg:hidden ${menuOpen ? 'opacity-100' : 'pointer-events-none opacity-0'}`}
      />
      <aside
        id="admin-navigation"
        className={`fixed inset-y-0 left-0 z-40 w-72 max-w-[85vw] shrink-0 overflow-y-auto overscroll-contain bg-navy-800 text-navy-100 shadow-xl transition-[transform,visibility] duration-200 ease-out [scrollbar-color:var(--color-navy-600)_transparent] lg:sticky lg:top-0 lg:z-auto lg:h-screen lg:w-64 lg:max-w-none lg:translate-x-0 lg:shadow-none lg:visible ${menuOpen ? 'visible translate-x-0' : 'invisible -translate-x-full'}`}
      >
        <div className="flex items-center justify-between border-b border-navy-700 py-3 pr-3 pl-5 lg:py-4">
          <span className="text-lg font-semibold tracking-tight text-white">Seafarers Crew</span>
          <button
            type="button"
            aria-label="Close menu"
            onClick={() => setMenuOpen(false)}
            className="inline-flex size-9 items-center justify-center rounded-md text-navy-100 hover:bg-navy-700 hover:text-white lg:hidden"
          >
            <Icon name="close" className="size-5" />
          </button>
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
          <IconButton
            icon="menu"
            label={menuOpen ? 'Close menu' : 'Open menu'}
            bordered
            aria-expanded={menuOpen}
            aria-controls="admin-navigation"
            onClick={() => setMenuOpen((o) => !o)}
            className="lg:hidden"
          />
          <div className="ml-auto flex min-w-0 items-center gap-1 text-sm sm:gap-3">
            <NavLink to="/account" className="min-w-0 truncate rounded-md px-2 py-1 text-slate-700 hover:text-navy-800">
              {admin?.name}
            </NavLink>
            <button
              type="button"
              onClick={() => logout.mutate()}
              disabled={logout.isPending}
              aria-label="Sign out"
              title="Sign out"
              className="inline-flex size-9 items-center justify-center rounded-md text-slate-600 hover:bg-slate-100 sm:w-auto sm:gap-2 sm:px-2"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="size-[1.125rem] sm:hidden">
                <path d="M10 4H5v16h5M15 8l4 4-4 4M19 12H9" />
              </svg>
              <span className="max-sm:sr-only">Sign out</span>
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
