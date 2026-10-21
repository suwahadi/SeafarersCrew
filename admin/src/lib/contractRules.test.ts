import { addMonths, isClosed, isFieldEditable, monthlyAllowance, totalMonthlyPay, type ContractField, type ContractStatus } from './contractRules'

describe('addMonths', () => {
  it.each([
    ['2026-01-15', 6, '2026-07-15'],
    ['2026-01-31', 1, '2026-02-28'],
    ['2028-01-31', 1, '2028-02-29'],
    ['2026-08-31', 6, '2027-02-28'],
    ['2026-10-31', 4, '2027-02-28'],
    ['2026-03-31', 1, '2026-04-30'],
    ['2026-11-30', 3, '2027-02-28'],
    ['2026-12-01', 12, '2027-12-01'],
  ])('adds months to %s (+%d) and clamps to month end: %s', (start, months, want) => {
    expect(addMonths(start, months)).toBe(want)
  })

  it('returns an empty string for invalid input', () => {
    expect(addMonths('', 6)).toBe('')
    expect(addMonths('2026-01-01', Number.NaN)).toBe('')
  })
})

describe('allowance preview', () => {
  it('uses the value for fixed monthly allowances', () => {
    expect(monthlyAllowance('fixed_month', '150', '1000', 2)).toBe('150.00')
  })

  it('multiplies per-day allowances by 30', () => {
    expect(monthlyAllowance('fixed_day', '5.50', '1000', 2)).toBe('165.00')
  })

  it('takes a percentage of the basic salary', () => {
    expect(monthlyAllowance('percent_basic', '12.5', '2000', 2)).toBe('250.00')
  })

  it('rounds half away from zero to the currency decimals', () => {
    expect(monthlyAllowance('percent_basic', '10', '1000.05', 2)).toBe('100.01')
    expect(monthlyAllowance('percent_basic', '3.5', '1234567', 0)).toBe('43210')
  })

  it('treats empty or invalid values as zero', () => {
    expect(monthlyAllowance('fixed_month', '', '1000', 2)).toBe('0.00')
    expect(monthlyAllowance('fixed_day', 'abc', '1000', 2)).toBe('0.00')
  })

  it('sums basic salary and allowances for the total monthly pay', () => {
    const total = totalMonthlyPay(
      '2000',
      [
        { basis: 'fixed_month', value: '100' },
        { basis: 'fixed_day', value: '2' },
        { basis: 'percent_basic', value: '10' },
      ],
      2,
    )
    expect(total).toBe('2360.00')
  })

  it('does not drift like floating point arithmetic', () => {
    expect(totalMonthlyPay('0.10', [{ basis: 'fixed_month', value: '0.20' }], 2)).toBe('0.30')
  })
})

describe('contract lock rules', () => {
  const all: ContractField[] = [
    'crew_id',
    'vessel_id',
    'rank_id',
    'principal_id',
    'employer_company_id',
    'sign_on_date',
    'duration_months',
    'planned_sign_off_date',
    'point_of_hire',
    'sign_on_port',
    'sign_off_port',
    'currency_code',
    'basic_salary',
    'working_hours_week',
    'leave_days_month',
    'probation_months',
    'notes',
    'allowances',
  ]

  it('allows every field for new and draft contracts', () => {
    for (const f of all) {
      expect(isFieldEditable(undefined, f)).toBe(true)
      expect(isFieldEditable('draft', f)).toBe(true)
    }
  })

  it('limits active contracts to the planned sign-off date, ports and notes', () => {
    const editable = all.filter((f) => isFieldEditable('active', f))
    expect(editable).toEqual(['planned_sign_off_date', 'sign_on_port', 'sign_off_port', 'notes'])
  })

  it.each<ContractStatus>(['completed', 'terminated'])('locks every field of a %s contract', (status) => {
    expect(all.some((f) => isFieldEditable(status, f))).toBe(false)
    expect(isClosed(status)).toBe(true)
  })

  it('does not treat draft or active as closed', () => {
    expect(isClosed('draft')).toBe(false)
    expect(isClosed('active')).toBe(false)
  })
})
