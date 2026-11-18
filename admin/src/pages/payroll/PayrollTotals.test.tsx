import { render } from '@testing-library/react'
import { PayrollTotals } from './PayrollTotals'

function parts(container: HTMLElement): string[] {
  return [...container.querySelectorAll('p > span')].map((s) => s.textContent.replace(/\s+/g, ' ').trim())
}

describe('PayrollTotals', () => {
  it('names each currency once and uses the singular for one row', () => {
    const { container } = render(<PayrollTotals totals={{ count: 1, usd: '1800.00', idr: '32100000.00' }} />)
    expect(parts(container)).toEqual(['1 row', 'Total USD 1,800.00', 'IDR 32,100,000'])
  })

  it('uses the plural for several rows', () => {
    const { container } = render(<PayrollTotals totals={{ count: 6, usd: '13700.80', idr: '240100000.00' }} />)
    expect(parts(container)).toEqual(['6 rows', 'Total USD 13,700.80', 'IDR 240,100,000'])
  })
})
