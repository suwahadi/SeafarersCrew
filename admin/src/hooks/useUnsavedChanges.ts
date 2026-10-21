import { useEffect, useRef } from 'react'
import { useBlocker } from 'react-router'

export function useUnsavedChanges(dirty: boolean) {
  const leaving = useRef(false)
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && !leaving.current && currentLocation.pathname !== nextLocation.pathname)

  useEffect(() => {
    if (!dirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [dirty])

  return {
    blocker,
    markLeaving: () => {
      leaving.current = true
    },
  }
}
