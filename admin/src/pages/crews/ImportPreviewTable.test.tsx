import { render, screen, within } from '@testing-library/react'
import type { ImportPreview } from '@/api/crews'
import { ImportPreviewTable } from './ImportPreviewTable'

const preview: ImportPreview = {
  token: 't',
  total: 3,
  valid: 1,
  invalid: 1,
  duplicate: 1,
  rows: [
    { line: 2, status: 'valid', values: { full_name: 'Budi Santoso', applied_rank: 'AB', passport_number: 'A123' } },
    { line: 3, status: 'invalid', values: { full_name: '', applied_rank: 'XX' }, errors: { full_name: 'The full name is required.', applied_rank: 'Unknown rank.' } },
    { line: 4, status: 'duplicate', values: { full_name: 'Ahmad', passport_number: 'B999' } },
  ],
}

describe('ImportPreviewTable', () => {
  it('shows the counts and one row per line with its status', () => {
    render(<ImportPreviewTable preview={preview} />)
    const summary = screen.getByLabelText('Validation summary')
    expect(within(summary).getByText('Rows:').nextElementSibling).toHaveTextContent('3')
    expect(within(summary).getByText('Valid:').nextElementSibling).toHaveTextContent('1')
    expect(screen.getAllByRole('row')).toHaveLength(4)
    expect(screen.getByText('Valid')).toBeInTheDocument()
    expect(screen.getByText('Invalid')).toBeInTheDocument()
    expect(screen.getByText('Duplicate (skipped)')).toBeInTheDocument()
  })

  it('lists the per-row validation errors by field', () => {
    render(<ImportPreviewTable preview={preview} />)
    expect(screen.getByText('The full name is required.', { exact: false })).toBeInTheDocument()
    expect(screen.getByText('Unknown rank.', { exact: false })).toBeInTheDocument()
  })
})
