import Link from 'next/link'
import { Icon } from '@/components/ui/icons'
import type { Crumb } from '@/lib/jsonld'

export function Breadcrumbs({ crumbs }: { crumbs: readonly Crumb[] }) {
  return (
    <nav aria-label="Breadcrumb" className="mb-4">
      <ol className="flex flex-wrap items-center gap-1 text-sm opacity-85">
        {crumbs.map((crumb, index) => {
          const last = index === crumbs.length - 1
          return (
            <li key={crumb.path} className="flex items-center gap-1">
              {last ? (
                <span aria-current="page" className="font-medium">
                  {crumb.name}
                </span>
              ) : (
                <>
                  <Link href={crumb.path} className="underline-offset-2 hover:underline">
                    {crumb.name}
                  </Link>
                  <Icon name="chevron-right" className="size-4" />
                </>
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
