import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { errorMessage, fieldErrors } from '@/api/client'
import { waitForJob } from '@/api/jobs'
import { payrollApi, payrollKeys, type PayrollImportReport, type PayrollPreview, type PayrollRowStatus } from '@/api/payroll'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/ui/Icon'
import { Badge, type Tone } from '@/components/ui/Layout'
import { Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { formatMoney } from '@/lib/format'

const status: Record<PayrollRowStatus, { tone: Tone; label: string }> = {
  valid: { tone: 'green', label: 'Valid' },
  new_account: { tone: 'blue', label: 'New account' },
  error: { tone: 'red', label: 'Error' },
}

const columnLabel: Record<string, string> = {
  employee_no: 'Employee ID',
  account_number: 'Account no.',
  bank_name: 'Bank',
  account_name: 'Account name',
  usd: 'USD',
  idr: 'IDR',
  date: 'Date',
  remark: 'Remark',
}

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`

function summaryText(p: PayrollPreview): string {
  return `${plural(p.total, 'row')}: ${p.valid} valid, ${plural(p.new_accounts, 'new account')}, ${plural(p.errors, 'error')} · replaces ${plural(p.will_replace, 'existing row')}`
}

interface Props {
  periodId: string
  onClose: () => void
}

export function PayrollImportDialog({ periodId, onClose }: Props) {
  const qc = useQueryClient()
  const toast = useToast()
  const input = useRef<HTMLInputElement>(null)
  const [preview, setPreview] = useState<PayrollPreview | null>(null)
  const [fileName, setFileName] = useState('')

  const validate = useMutation({
    mutationFn: (file: File) => payrollApi.validateImport(periodId, file),
    onMutate: () => setPreview(null),
    onSuccess: setPreview,
  })

  const commit = useMutation({
    mutationFn: async (token: string) => {
      const queued = await payrollApi.commitImport(periodId, token)
      const job = await waitForJob(queued.id)
      if (job.status === 'failed') throw new Error(job.error ?? 'The import failed.')
      return job.result as PayrollImportReport | undefined
    },
    onSuccess: (report) => {
      void qc.invalidateQueries({ queryKey: ['grid', 'payroll-lines'] })
      void qc.invalidateQueries({ queryKey: ['grid', 'payroll-periods'] })
      void qc.invalidateQueries({ queryKey: payrollKeys.period(periodId) })
      void qc.invalidateQueries({ queryKey: payrollKeys.facets(periodId) })
      toast.success(`Imported ${plural(report?.lines ?? 0, 'row')}, ${plural(report?.accounts_added ?? 0, 'new bank account')}.`)
      onClose()
    },
  })

  const removed = preview ? preview.will_replace - preview.total : 0
  const canCommit = preview !== null && preview.errors === 0 && preview.token !== '' && !validate.isPending
  const failure = validate.error ?? commit.error

  return (
    <Modal
      open
      size="xl"
      title="Import salary sheet"
      onClose={onClose}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={commit.isPending}>
            Cancel
          </Button>
          <Button onClick={() => preview && commit.mutate(preview.token)} disabled={!canCommit || commit.isPending} loading={commit.isPending}>
            Confirm import
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <p className="text-sm text-slate-600">
          Upload the bank-transfer workbook (.xlsx) as it is. After you confirm, its rows replace every row in this period. Accounts that a crew member does not have yet are added to their
          profile.
        </p>
        <div className="flex flex-wrap items-center gap-3">
          <input
            ref={input}
            type="file"
            accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
            aria-label="Excel file"
            className="sr-only"
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) {
                setFileName(file.name)
                validate.mutate(file)
              }
              e.target.value = ''
            }}
          />
          <Button variant="secondary" icon={<Icon name="upload" />} className="max-sm:min-h-10" loading={validate.isPending} onClick={() => input.current?.click()}>
            {preview ? 'Choose another file' : 'Choose Excel file'}
          </Button>
          {fileName ? <span className="min-w-0 text-sm break-all text-slate-700">{fileName}</span> : null}
        </div>
        {failure ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {fieldErrors(failure).file ?? errorMessage(failure)}
          </p>
        ) : null}
        {preview ? (
          <div className="space-y-3">
            <p aria-label="Validation summary" className="text-sm font-medium text-navy-800">
              {summaryText(preview)}
            </p>
            {preview.errors > 0 ? (
              <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
                Fix the rows marked Error in the file and upload it again. Nothing has been changed.
              </p>
            ) : null}
            {removed > 0 ? (
              <p role="alert" className="rounded-md bg-yellow-50 p-3 text-sm text-yellow-800">
                {removed === 1 ? '1 existing row will be removed.' : `${removed} existing rows will be removed.`}
              </p>
            ) : null}
            <div className="max-h-[24rem] overflow-auto rounded-md ring-1 ring-slate-200">
              <table className="min-w-full divide-y divide-slate-200 text-sm">
                <thead className="sticky top-0 bg-slate-50">
                  <tr>
                    {['Line', 'Status', 'Employee ID', 'Name', 'Bank', 'Account no.', 'IDR', 'Notes'].map((h) => (
                      <th key={h} scope="col" className="px-3 py-2 text-left text-xs font-semibold whitespace-nowrap text-slate-600 uppercase">
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {preview.rows.map((r) => (
                    <tr key={r.line} className={r.status === 'error' ? 'bg-red-50/40' : undefined}>
                      <td className="px-3 py-1.5 font-mono">{r.line}</td>
                      <td className="px-3 py-1.5 whitespace-nowrap">
                        <Badge tone={status[r.status].tone}>{status[r.status].label}</Badge>
                      </td>
                      <td className="px-3 py-1.5 font-mono">{r.employee_no}</td>
                      <td className="px-3 py-1.5 whitespace-nowrap">{r.crew_name}</td>
                      <td className="px-3 py-1.5 whitespace-nowrap">{r.bank_name}</td>
                      <td className="px-3 py-1.5 font-mono whitespace-nowrap">{r.account_number}</td>
                      <td className="px-3 py-1.5 text-right whitespace-nowrap tabular-nums">{r.amount_idr ? formatMoney(r.amount_idr, 'IDR', 0) : ''}</td>
                      <td className="min-w-64 px-3 py-1.5 text-xs">
                        <ul className="space-y-0.5">
                          {Object.entries(r.errors ?? {}).map(([field, msg]) => (
                            <li key={field} className="text-red-700">
                              <span className="font-semibold">{columnLabel[field] ?? field}:</span> <span>{msg}</span>
                            </li>
                          ))}
                          {(r.warnings ?? []).map((w) => (
                            <li key={w} className="text-yellow-800">
                              <span>{w}</span>
                            </li>
                          ))}
                        </ul>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        ) : null}
      </div>
    </Modal>
  )
}
