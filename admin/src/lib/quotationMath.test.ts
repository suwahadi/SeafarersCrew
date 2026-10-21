import { lineUnits, quotationTotals } from './quotationMath'

describe('quotationTotals', () => {
  it('multiplies crew count, daily rate and days and adds lump sums', () => {
    const t = quotationTotals(
      [
        { line_type: 'daily_rate', crew_count: '2', daily_rate: '150.50', days: '30' },
        { line_type: 'lump_sum', amount: '1000' },
      ],
      '0',
      '0',
      2,
    )
    expect(t.lines).toEqual(['9030.00', '1000.00'])
    expect(t.subtotal).toBe('10030.00')
    expect(t.grandTotal).toBe('10030.00')
  })

  it('applies tax to the subtotal less the discount', () => {
    const t = quotationTotals([{ line_type: 'lump_sum', amount: '1000' }], '10', '100', 2)
    expect(t.tax).toBe('90.00')
    expect(t.grandTotal).toBe('990.00')
  })

  it('rounds tax to the currency decimals', () => {
    const t = quotationTotals([{ line_type: 'lump_sum', amount: '99.99' }], '7.5', '0', 2)
    expect(t.tax).toBe('7.50')
    expect(t.grandTotal).toBe('107.49')
  })

  it('supports currencies without decimals', () => {
    const t = quotationTotals([{ line_type: 'daily_rate', crew_count: '3', daily_rate: '1500000', days: '10' }], '11', '500000', 0)
    expect(t.subtotal).toBe('45000000')
    expect(t.tax).toBe('4895000')
    expect(t.grandTotal).toBe('49395000')
  })

  it('flags a discount larger than the subtotal', () => {
    expect(quotationTotals([{ line_type: 'lump_sum', amount: '10' }], '0', '20', 2).discountExceedsSubtotal).toBe(true)
  })

  it('ignores incomplete lines', () => {
    expect(lineUnits({ line_type: 'daily_rate', crew_count: '', daily_rate: '10', days: '5' }, 2)).toBe(0)
    expect(lineUnits({ line_type: 'daily_rate', crew_count: '1.5', daily_rate: '10', days: '5' }, 2)).toBe(0)
  })
})
