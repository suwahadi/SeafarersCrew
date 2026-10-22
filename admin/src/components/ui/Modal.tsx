import { useEffect, useRef, type ReactNode } from 'react'
import { Button } from './Button'

interface ModalProps {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
  size?: 'md' | 'lg' | 'xl'
}

const widths = { md: 'max-w-lg', lg: 'max-w-2xl', xl: 'max-w-5xl' }

export function Modal({ open, title, onClose, children, footer, size = 'md' }: ModalProps) {
  const ref = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  }, [open])

  return (
    <dialog
      ref={ref}
      aria-labelledby="modal-title"
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      className={`m-auto w-[calc(100%-1rem)] ${widths[size]} max-h-[92dvh] rounded-lg p-0 shadow-xl backdrop:bg-slate-900/40 open:flex open:flex-col sm:w-full sm:max-h-[90dvh]`}
    >
      {open ? (
        <>
          <header className="flex items-center justify-between gap-2 border-b border-slate-200 px-4 py-3 sm:px-5">
            <h2 id="modal-title" className="text-base font-semibold text-navy-800">
              {title}
            </h2>
            <button type="button" onClick={onClose} aria-label="Close" className="inline-flex size-9 shrink-0 items-center justify-center rounded text-slate-500 hover:bg-slate-100">
              ✕
            </button>
          </header>
          <div className="min-h-0 overflow-y-auto px-4 py-4 sm:px-5">{children}</div>
          {footer ? <footer className="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-4 py-3 sm:px-5">{footer}</footer> : null}
        </>
      ) : null}
    </dialog>
  )
}

interface ConfirmProps {
  open: boolean
  title: string
  message: ReactNode
  confirmLabel?: string
  danger?: boolean
  loading?: boolean
  onConfirm: () => void
  onClose: () => void
}

export function ConfirmDialog({ open, title, message, confirmLabel = 'Confirm', danger = false, loading = false, onConfirm, onClose }: ConfirmProps) {
  return (
    <Modal
      open={open}
      title={title}
      onClose={onClose}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button variant={danger ? 'danger' : 'primary'} onClick={onConfirm} loading={loading}>
            {confirmLabel}
          </Button>
        </>
      }
    >
      <div className="text-sm text-slate-700">{message}</div>
    </Modal>
  )
}
