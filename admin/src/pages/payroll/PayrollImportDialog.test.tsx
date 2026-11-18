import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { PayrollPreview, PayrollPreviewRow } from '@/api/payroll'
import { mockApi, renderApp } from '@/test/utils'
import { PayrollImportDialog } from './PayrollImportDialog'

function row(line: number, status: PayrollPreviewRow['status'], extra: Partial<PayrollPreviewRow> = {}): PayrollPreviewRow {
  return {
    line,
    status,
    employee_no: '900101',
    crew_name: 'Budi Santoso',
    bank_name: 'BANK NEGARA INDONESIA',
    account_name: 'BUDI SANTOSO',
    account_number: '3200000001',
    amount_usd: '1800.00',
    amount_idr: '32100000.00',
    transfer_date: '2026-10-20',
    remark: '',
    ...extra,
  }
}

function preview(extra: Partial<PayrollPreview>): PayrollPreview {
  return {
    token: 'tok',
    total: 2,
    valid: 2,
    new_accounts: 0,
    errors: 0,
    with_warnings: 0,
    will_replace: 0,
    rows: [row(10, 'valid'), row(11, 'valid')],
    ...extra,
  }
}

const file = () =>
  new File(['xlsx'], 'Bank Transfer Salary - September 2026.xlsx', {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  })

async function upload(reply: PayrollPreview) {
  const calls = mockApi((call) => {
    if (call.path.endsWith('/import/validate')) return { body: { data: reply } }
    if (call.path.endsWith('/import/commit'))
      return {
        status: 202,
        body: {
          data: {
            id: 'job-1',
            kind: 'payroll_import',
            status: 'queued',
            attempts: 0,
            created_at: '',
          },
        },
      }
    if (call.path === '/jobs/job-1')
      return {
        body: {
          data: {
            id: 'job-1',
            kind: 'payroll_import',
            status: 'succeeded',
            attempts: 1,
            created_at: '',
            result: { lines: 91, accounts_added: 2 },
          },
        },
      }
    return undefined
  })
  const onClose = vi.fn()
  const user = userEvent.setup()
  renderApp(<PayrollImportDialog periodId="PERAAAAA2" onClose={onClose} />)
  await user.upload(screen.getByLabelText('Excel file'), file())
  return { calls, onClose, user }
}

describe('PayrollImportDialog', () => {
  it('shows row statuses and blocks confirm while errors exist', async () => {
    await upload(
      preview({
        token: '',
        valid: 1,
        errors: 1,
        rows: [
          row(10, 'valid'),
          row(11, 'error', {
            employee_no: '999999',
            crew_name: '',
            errors: { employee_no: 'No crew member has this Employee ID.' },
          }),
        ],
      }),
    )
    expect(await screen.findByText('No crew member has this Employee ID.')).toBeInTheDocument()
    expect(screen.getByText('Error')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirm import' })).toBeDisabled()
    expect(screen.getByText(/Fix the rows marked Error/)).toBeInTheDocument()
  })

  it('warns when the file has fewer rows than the period', async () => {
    await upload(preview({ total: 89, valid: 89, will_replace: 91 }))
    expect(await screen.findByText('2 existing rows will be removed.')).toBeInTheDocument()
  })

  it('marks new accounts and warnings', async () => {
    await upload(
      preview({
        valid: 1,
        new_accounts: 1,
        with_warnings: 1,
        rows: [
          row(10, 'valid'),
          row(11, 'new_account', {
            warnings: ['No contract covers this period; rank and vessel are left empty.'],
          }),
        ],
      }),
    )
    expect(await screen.findByText('New account')).toBeInTheDocument()
    expect(screen.getByText('No contract covers this period; rank and vessel are left empty.')).toBeInTheDocument()
    expect(screen.getByText('2 rows: 1 valid, 1 new account, 0 errors · replaces 0 existing rows')).toBeInTheDocument()
  })

  it('commits and reports the result', async () => {
    const { calls, onClose, user } = await upload(preview({ will_replace: 2 }))
    const confirm = await screen.findByRole('button', {
      name: 'Confirm import',
    })
    expect(confirm).toBeEnabled()
    await user.click(confirm)

    expect(await screen.findByText('Imported 91 rows, 2 new bank accounts.')).toBeInTheDocument()
    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(calls.find((c) => c.path.endsWith('/import/commit'))?.body).toEqual({
      token: 'tok',
    })
  })

  it('shows the file-level reason when the workbook is rejected', async () => {
    mockApi(() => ({
      status: 422,
      body: { error: { code: 'validation_failed', message: 'The given data is invalid.', fields: { file: 'The file has no data rows.' } } },
    }))
    const user = userEvent.setup()
    renderApp(<PayrollImportDialog periodId="PERAAAAA2" onClose={vi.fn()} />)
    await user.upload(screen.getByLabelText('Excel file'), file())
    expect(await screen.findByText('The file has no data rows.')).toBeInTheDocument()
    expect(screen.queryByText('The given data is invalid.')).not.toBeInTheDocument()
  })

  it('sends one commit even when Confirm is clicked again while the import runs', async () => {
    let polls = 0
    const calls = mockApi((call) => {
      if (call.path.endsWith('/import/validate')) return { body: { data: preview({}) } }
      if (call.path.endsWith('/import/commit')) return { status: 202, body: { data: { id: 'job-1', kind: 'payroll_import', status: 'queued', attempts: 0, created_at: '' } } }
      if (call.path === '/jobs/job-1') {
        polls++
        const done = polls > 1
        return {
          body: {
            data: { id: 'job-1', kind: 'payroll_import', status: done ? 'succeeded' : 'running', attempts: 1, created_at: '', result: done ? { lines: 2, accounts_added: 0 } : undefined },
          },
        }
      }
      return undefined
    })
    const user = userEvent.setup()
    renderApp(<PayrollImportDialog periodId="PERAAAAA2" onClose={vi.fn()} />)
    await user.upload(screen.getByLabelText('Excel file'), file())
    const confirm = await screen.findByRole('button', { name: /Confirm import/ })
    await user.click(confirm)
    await waitFor(() => expect(confirm).toBeDisabled())
    await user.click(confirm)
    expect(await screen.findByText('Imported 2 rows, 0 new bank accounts.')).toBeInTheDocument()
    expect(calls.filter((c) => c.path.endsWith('/import/commit'))).toHaveLength(1)
  })
})
