import Link from 'next/link'
import { Icon } from '@/components/ui/icons'
import { pageHref, pageItems } from '@/lib/pagination'

interface PaginationProps {
  basePath: string
  current: number
  pages: number
  label?: string
}

const ITEM =
  'inline-flex min-w-10 items-center justify-center gap-1 rounded-md border border-line px-3 py-2 text-sm font-medium text-primary transition-colors duration-150 ease-out hover:bg-surface'

export function Pagination({ basePath, current, pages, label = 'Pagination' }: PaginationProps) {
  if (pages <= 1) {
    return null
  }
  return (
    <nav aria-label={label} className="mt-10">
      <ul className="flex flex-wrap items-center justify-center gap-2">
        <li>
          {current > 1 ? (
            <Link href={pageHref(basePath, current - 1)} rel="prev" className={ITEM}>
              <Icon name="chevron-left" className="size-4" />
              Previous
            </Link>
          ) : null}
        </li>
        {pageItems(current, pages).map((item, index) =>
          item === 'gap' ? (
            <li key={`gap-${String(index)}`} aria-hidden="true" className="px-1 text-muted">
              &hellip;
            </li>
          ) : (
            <li key={item}>
              <Link
                href={pageHref(basePath, item)}
                aria-current={item === current ? 'page' : undefined}
                aria-label={`Page ${String(item)}`}
                className={
                  item === current ? `${ITEM} border-primary bg-primary text-on-primary hover:bg-primary-hover` : ITEM
                }
              >
                {item}
              </Link>
            </li>
          ),
        )}
        <li>
          {current < pages ? (
            <Link href={pageHref(basePath, current + 1)} rel="next" className={ITEM}>
              Next
              <Icon name="chevron-right" className="size-4" />
            </Link>
          ) : null}
        </li>
      </ul>
    </nav>
  )
}
