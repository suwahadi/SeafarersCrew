import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { PayrollLine } from '@/api/payroll'
import { mockApi, renderApp } from '@/test/utils'
import { LineFormDialog } from './LineFormDialog'

const line: PayrollLine = {
  id: '0f0f0f0f-0000-4000-8000-000000000001',
  line_no: 1,
  crew: { id: 'CREWAAAA2', name: 'Budi Santoso' },
  employee_no: '900101',
  crew_name: 'Budi Santoso',
  rank_code: '2E',
  vessel_name: 'MV. NUSA 52',
  bank_account_id: 'acc-bni',
  bank_name: 'BANK NEGARA INDONESIA',
  account_name: 'BUDI SANTOSO',
  account_number: '3200000001',
  amount_usd: '1800.00',
  amount_idr: '32100000.00',
  transfer_date: '2026-10-20',
  remark: '',
}

const accounts = [
  {
    id: 'acc-bni',
    bank_name: 'BANK NEGARA INDONESIA',
    account_name: 'BUDI SANTOSO',
    account_number: '3200000001',
    currency_code: 'IDR',
    cover_bank: '',
    remarks: '',
    is_primary: true,
  },
  {
    id: 'acc-mandiri',
    bank_name: 'MANDIRI',
    account_name: 'SITI RAHAYU',
    account_number: '0123456789',
    currency_code: 'IDR',
    cover_bank: '',
    remarks: '',
    is_primary: false,
  },
]

describe('LineFormDialog', () => {
  it('requires crew, account and IDR', async () => {
    mockApi(() => undefined)
    const user = userEvent.setup()
    renderApp(<LineFormDialog periodId="PERAAAAA2" onClose={() => undefined} />)

    await user.click(screen.getByRole('button', { name: 'Save row' }))
    expect(await screen.findByText('Select the crew member.')).toBeInTheDocument()
    expect(screen.getByText('Select the bank account.')).toBeInTheDocument()
    expect(screen.getByText('IDR is required.')).toBeInTheDocument()
  })

  it("lists only the crew member's accounts and saves the row", async () => {
    const calls = mockApi((call) => {
      if (call.path === '/crews/CREWAAAA2/bank-accounts') return { body: { data: accounts } }
      if (call.method === 'PUT') return { body: { data: line } }
      return undefined
    })
    const onClose = vi.fn()
    const user = userEvent.setup()
    renderApp(<LineFormDialog periodId="PERAAAAA2" line={line} onClose={onClose} />)

    const select = await screen.findByLabelText(/Bank account/)
    await waitFor(() =>
      expect(
        within(select)
          .getAllByRole('option')
          .map((o) => o.textContent),
      ).toContain('MANDIRI · 0123456789 · SITI RAHAYU'),
    )
    expect(within(select).getAllByRole('option')).toHaveLength(3)

    await user.selectOptions(select, 'acc-mandiri')
    const idr = screen.getByLabelText(/IDR/)
    await user.clear(idr)
    await user.type(idr, '8000000')
    await user.click(screen.getByRole('button', { name: 'Save row' }))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    const put = calls.find((c) => c.method === 'PUT')
    expect(put?.path).toBe(`/payroll-lines/${line.id}`)
    expect(put?.body).toMatchObject({
      crew_id: 'CREWAAAA2',
      bank_account_id: 'acc-mandiri',
      amount_idr: '8000000',
      amount_usd: '1800.00',
    })
  })

  it('rejects a negative USD', async () => {
    mockApi((call) => (call.path === '/crews/CREWAAAA2/bank-accounts' ? { body: { data: accounts } } : undefined))
    const user = userEvent.setup()
    renderApp(<LineFormDialog periodId="PERAAAAA2" line={line} onClose={() => undefined} />)

    const usd = await screen.findByLabelText(/USD/)
    await user.clear(usd)
    await user.type(usd, '-1')
    await user.click(screen.getByRole('button', { name: 'Save row' }))
    expect(await screen.findByText('USD cannot be negative.')).toBeInTheDocument()
  })

  it("shows the row's current account once the accounts load", async () => {
    mockApi((call) => (call.path === '/crews/CREWAAAA2/bank-accounts' ? { body: { data: accounts } } : undefined))
    renderApp(<LineFormDialog periodId="PERAAAAA2" line={{ ...line, bank_account_id: 'acc-mandiri' }} onClose={() => undefined} />)
    const select = await screen.findByLabelText(/Bank account/)
    await waitFor(() => expect(select).toHaveValue('acc-mandiri'))
  })
})
