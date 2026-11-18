import type { PayrollTotals as Totals } from '@/api/payroll'
import { formatMoney } from '@/lib/format'

export function PayrollTotals({ totals }: { totals: Totals }) {
  return (
    <p className="flex flex-wrap gap-x-5 gap-y-1">
      <span>
        <strong className="text-navy-800">{totals.count}</strong> {totals.count === 1 ? 'row' : 'rows'}
      </span>
      <span>
        Total <strong className="text-navy-800 tabular-nums">{formatMoney(totals.usd, 'USD')}</strong>
      </span>
      <span>
        <strong className="text-navy-800 tabular-nums">{formatMoney(totals.idr, 'IDR', 0)}</strong>
      </span>
    </p>
  )
}
