import type { ReactNode } from 'react'
import { Spinner } from '@/components/ui/Spinner'

export interface RowColumn<T> {
  header: string
  cell: (row: T) => ReactNode
  className?: string
}

interface RowTableProps<T> {
  rows: T[] | undefined
  columns: RowColumn<T>[]
  rowKey: (row: T) => string
  actions?: (row: T) => ReactNode
  empty: string
  loading?: boolean
  caption: string
}

export function RowTable<T>({ rows, columns, rowKey, actions, empty, loading, caption }: RowTableProps<T>) {
  if (loading) {
    return (
      <div className="flex justify-center p-6 text-navy-800">
        <Spinner label={`Loading ${caption.toLowerCase()}`} />
      </div>
    )
  }
  return (
    <div className="overflow-x-auto rounded-md ring-1 ring-slate-200">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <caption className="sr-only">{caption}</caption>
        <thead className="bg-slate-50">
          <tr>
            {columns.map((c) => (
              <th key={c.header} scope="col" className="px-3 py-2 text-left text-xs font-semibold tracking-wide text-slate-600 uppercase whitespace-nowrap">
                {c.header}
              </th>
            ))}
            {actions ? (
              <th scope="col" className="px-3 py-2 text-right text-xs font-semibold tracking-wide text-slate-600 uppercase">
                Actions
              </th>
            ) : null}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 bg-white">
          {!rows || rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length + (actions ? 1 : 0)} className="p-6 text-center text-slate-500">
                {empty}
              </td>
            </tr>
          ) : (
            rows.map((r) => (
              <tr key={rowKey(r)}>
                {columns.map((c) => (
                  <td key={c.header} className={`px-3 py-2 whitespace-nowrap ${c.className ?? ''}`}>
                    {c.cell(r)}
                  </td>
                ))}
                {actions ? <td className="px-3 py-2 text-right whitespace-nowrap">{actions(r)}</td> : null}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}
