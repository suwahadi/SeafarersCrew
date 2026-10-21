import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { authApi } from '@/api/auth'
import { errorMessage } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog, Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'

export function codesToText(codes: string[]): string {
  return `Seafarers Crew admin recovery codes\nEach code can be used once.\n\n${codes.join('\n')}\n`
}

export function RecoveryCodesList({ codes, onDone }: { codes: string[]; onDone: () => void }) {
  const toast = useToast()

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(codes.join('\n'))
      toast.success('Recovery codes copied.')
    } catch {
      toast.error('Copying failed. Select the codes and copy them manually.')
    }
  }

  const download = () => {
    const url = URL.createObjectURL(new Blob([codesToText(codes)], { type: 'text/plain' }))
    const link = document.createElement('a')
    link.href = url
    link.download = 'recovery-codes.txt'
    link.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="space-y-4">
      <p role="alert" className="rounded-md bg-yellow-50 p-3 text-sm text-yellow-800">
        These codes are shown only once. Store them somewhere safe. Any earlier codes no longer work.
      </p>
      <ol aria-label="Recovery codes" className="grid grid-cols-2 gap-2 font-mono text-sm">
        {codes.map((code) => (
          <li key={code} className="rounded bg-slate-100 px-3 py-1.5">
            {code}
          </li>
        ))}
      </ol>
      <div className="flex flex-wrap justify-end gap-2">
        <Button variant="secondary" onClick={() => void copy()}>
          Copy
        </Button>
        <Button variant="secondary" onClick={download}>
          Download .txt
        </Button>
        <Button onClick={onDone}>I have saved these codes</Button>
      </div>
    </div>
  )
}

export function RecoveryCodesPanel() {
  const toast = useToast()
  const [confirming, setConfirming] = useState(false)
  const [codes, setCodes] = useState<string[] | null>(null)

  const regenerate = useMutation({
    mutationFn: () => authApi.regenerateRecoveryCodes(),
    onSuccess: (res) => {
      setConfirming(false)
      setCodes(res.codes)
    },
    onError: (err) => {
      setConfirming(false)
      toast.error(errorMessage(err))
    },
  })

  return (
    <div className="space-y-2">
      <p className="text-sm text-slate-600">Recovery codes let you sign in if you forget your password. Regenerating replaces all existing codes.</p>
      <Button variant="secondary" onClick={() => setConfirming(true)}>
        Regenerate recovery codes
      </Button>
      <ConfirmDialog
        open={confirming}
        title="Regenerate recovery codes"
        message="Your existing recovery codes will stop working immediately. Continue?"
        confirmLabel="Regenerate"
        danger
        loading={regenerate.isPending}
        onConfirm={() => regenerate.mutate()}
        onClose={() => setConfirming(false)}
      />
      <Modal open={codes !== null} title="Your new recovery codes" onClose={() => setCodes(null)}>
        {codes ? <RecoveryCodesList codes={codes} onDone={() => setCodes(null)} /> : null}
      </Modal>
    </div>
  )
}
