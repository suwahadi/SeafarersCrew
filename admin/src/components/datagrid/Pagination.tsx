export const PAGE_SIZES = [10, 20, 30, 50, 100]

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
    <nav aria-label="Pagination" className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 px-3 py-2 text-sm text-slate-600">
      <span>
        {from}–{to} of {total.toLocaleString('en-US')}
      </span>
      <div className="flex items-center gap-2">
        <label className="flex items-center gap-1">
          Rows
          <select value={perPage} onChange={(e) => onPerPage(Number(e.target.value))} className="h-8 rounded-md border-0 text-sm ring-1 ring-slate-300">
            {PAGE_SIZES.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </label>
        <button type="button" disabled={page <= 1} onClick={() => onPage(page - 1)} className="rounded px-2 py-1 hover:bg-slate-100 disabled:opacity-40">
          Previous
        </button>
        <span>
          {page} / {pages}
        </span>
        <button type="button" disabled={page >= pages} onClick={() => onPage(page + 1)} className="rounded px-2 py-1 hover:bg-slate-100 disabled:opacity-40">
          Next
        </button>
      </div>
    </nav>
  )
}
