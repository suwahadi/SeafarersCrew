import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router'

export function useUrlFilters<K extends string>(keys: readonly K[]): [Record<K, string>, (key: K, value: string) => void, () => void] {
  const [params, setParams] = useSearchParams()
  const filters = useMemo(() => Object.fromEntries(keys.map((k) => [k, params.get(k) ?? ''])) as Record<K, string>, [keys, params])

  const set = useCallback(
    (key: K, value: string) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          if (value === '') next.delete(key)
          else next.set(key, value)
          return next
        },
        { replace: true },
      )
    },
    [setParams],
  )

  const clear = useCallback(() => {
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        for (const k of keys) next.delete(k)
        return next
      },
      { replace: true },
    )
  }, [keys, setParams])

  return [filters, set, clear]
}
