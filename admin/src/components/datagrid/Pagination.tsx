import { Icon } from '../ui/Icon'

export const PAGE_SIZES = [10, 20, 30, 50, 100]

const pagerButton =
  'inline-flex h-8 min-w-8 items-center justify-center gap-1 rounded-md bg-white px-2 sm:px-3 text-slate-700 ring-1 ring-inset ring-slate-300 transition-colors duration-150 hover:bg-slate-50 disabled:cursor-not-allowed disabled:text-slate-400 disabled:ring-slate-200 disabled:hover:bg-white'

interface PaginationProps {
  page: number
  perPage: number
  total: number
  onPage: (page: number) => void
  onPerPage: (perPage: number) => void
}

export function Pagination({ page, perPage, total, onPage, onPerPage }: PaginationProps) {
  const pages = Math.max(1, Math.ceil(total / perPage))
  const from = total === 0 ? 0 : (page - 1) * perPage + 1
  const to = Math.min(total, page * perPage)
  return (
    <nav aria-label="Pagination" className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 border-t border-slate-200 px-3 py-2.5 text-sm text-slate-600">
      <span className="tabular-nums">
        {from}–{to} of {total.toLocaleString('en-US')}
      </span>
      <div className="flex items-center gap-2 sm:gap-3">
        <label className="flex items-center gap-2">
          <span className="max-sm:sr-only">Rows</span>
          <select value={perPage} onChange={(e) => onPerPage(Number(e.target.value))} className="h-8 rounded-md border-0 bg-white text-sm text-slate-900 ring-1 ring-inset ring-slate-300">
            {PAGE_SIZES.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </label>
        <div className="flex items-center gap-1">
          <button type="button" disabled={page <= 1} onClick={() => onPage(page - 1)} className={pagerButton} aria-label="Previous">
            <Icon name="chevronLeft" />
            <span className="max-sm:sr-only">Previous</span>
          </button>
          <span className="px-1 tabular-nums sm:px-2">
            {page} / {pages}
          </span>
          <button type="button" disabled={page >= pages} onClick={() => onPage(page + 1)} className={pagerButton} aria-label="Next">
            <span className="max-sm:sr-only">Next</span>
            <Icon name="chevronRight" />
          </button>
        </div>
      </div>
    </nav>
  )
}
