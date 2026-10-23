'use client'

import { createContext, useCallback, useContext, useEffect, useRef, useState, type KeyboardEvent, type ReactNode, type SyntheticEvent } from 'react'
import { ApplicationForm, type ApplicationFormProps, type ApplicationStatus } from '@/components/forms/ApplicationForm'
import { buttonClass } from '@/components/ui/button'
import { Icon } from '@/components/ui/icons'

const ApplyContext = createContext<(() => void) | null>(null)

const DEEP_LINK = '#apply'

interface ApplyProviderProps {
  vacancyTitle: string
  vacancyId: string
  form: Omit<ApplicationFormProps, 'onStatusChange'>
  children: ReactNode
}

function keepOpen(event: SyntheticEvent): void {
  event.preventDefault()
}

function ignoreEscape(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    event.preventDefault()
  }
}

export function ApplyProvider({ vacancyTitle, vacancyId, form, children }: ApplyProviderProps) {
  const [open, setOpen] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [status, setStatus] = useState<ApplicationStatus>('editing')
  const dialog = useRef<HTMLDialogElement>(null)
  const confirm = useRef<HTMLDialogElement>(null)
  const heading = useRef<HTMLHeadingElement>(null)
  const closeButton = useRef<HTMLButtonElement>(null)
  const keepEditingButton = useRef<HTMLButtonElement>(null)
  const trigger = useRef<HTMLElement | null>(null)
  const dialogWanted = useRef(false)
  const confirmWanted = useRef(false)

  const show = useCallback(() => {
    trigger.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    dialogWanted.current = true
    setStatus('editing')
    setOpen(true)
  }, [])

  useEffect(() => {
    const element = dialog.current
    if (!open || !element) {
      return
    }
    element.setAttribute('closedby', 'none')
    if (!element.open) {
      element.showModal()
    }
    heading.current?.focus()
  }, [open])

  useEffect(() => {
    const element = confirm.current
    if (!confirming || !element) {
      return
    }
    if (!element.open) {
      element.showModal()
    }
    keepEditingButton.current?.focus()
  }, [confirming])

  useEffect(() => {
    function openFromAddress(): void {
      if (window.location.hash !== DEEP_LINK) {
        return
      }
      window.history.replaceState(null, '', window.location.pathname + window.location.search)
      show()
    }
    const frame = window.requestAnimationFrame(openFromAddress)
    window.addEventListener('hashchange', openFromAddress)
    return () => {
      window.cancelAnimationFrame(frame)
      window.removeEventListener('hashchange', openFromAddress)
    }
  }, [show])

  function finish(): void {
    confirmWanted.current = false
    dialogWanted.current = false
    confirm.current?.close()
    dialog.current?.close()
    setConfirming(false)
    setOpen(false)
    trigger.current?.focus()
  }

  function requestClose(): void {
    if (status === 'sent') {
      finish()
      return
    }
    confirmWanted.current = true
    setConfirming(true)
  }

  function keepEditing(): void {
    confirmWanted.current = false
    if (confirm.current?.open) {
      confirm.current.close()
    }
    setConfirming(false)
    closeButton.current?.focus()
  }

  return (
    <ApplyContext.Provider value={show}>
      {children}
      <dialog
        ref={dialog}
        aria-labelledby="apply-dialog-title"
        aria-describedby="apply-dialog-note"
        onCancel={keepOpen}
        onKeyDown={ignoreEscape}
        onClose={() => {
          if (dialogWanted.current && dialog.current && !dialog.current.open) {
            dialog.current.showModal()
          }
        }}
        className="apply-dialog m-0 h-dvh max-h-none w-full max-w-none overflow-hidden border-0 bg-white p-0 text-ink backdrop:bg-primary/60 open:flex open:flex-col sm:m-auto sm:h-auto sm:max-h-[90dvh] sm:w-[calc(100%-2rem)] sm:max-w-3xl sm:rounded-lg sm:shadow-2xl"
      >
        {open ? (
          <>
            <header className="flex shrink-0 items-start justify-between gap-4 border-b border-line px-5 py-4 sm:px-8 sm:py-5">
              <div className="min-w-0">
                <h2 id="apply-dialog-title" ref={heading} tabIndex={-1} className="text-xl font-bold text-primary focus:outline-none sm:text-2xl">
                  Apply for {vacancyTitle}
                </h2>
                <p className="mt-1 text-sm text-muted">
                  Vacancy <span translate="no">{vacancyId}</span>
                </p>
              </div>
              <button
                ref={closeButton}
                type="button"
                onClick={requestClose}
                disabled={status === 'sending'}
                aria-label="Close application form"
                title={status === 'sending' ? 'Wait until your application is sent' : 'Close application form'}
                className="-mr-2 inline-flex size-11 shrink-0 items-center justify-center rounded-md text-primary transition-colors duration-150 ease-out hover:bg-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-text disabled:cursor-not-allowed disabled:opacity-40"
              >
                <Icon name="close" className="size-6" />
              </button>
            </header>
            <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 py-6 sm:px-8 sm:py-8">
              <p id="apply-dialog-note" className="mb-6 text-sm text-muted">
                Fields marked with an asterisk are required. Your details are used only to process your application.
              </p>
              <ApplicationForm {...form} onStatusChange={setStatus} />
            </div>
          </>
        ) : null}
      </dialog>
      <dialog
        ref={confirm}
        role="alertdialog"
        aria-labelledby="apply-confirm-title"
        aria-describedby="apply-confirm-text"
        onCancel={(event) => {
          event.preventDefault()
          keepEditing()
        }}
        onClose={() => {
          if (confirmWanted.current) {
            keepEditing()
          }
        }}
        className="apply-dialog m-auto w-[calc(100%-2rem)] max-w-md rounded-lg border-0 bg-white p-6 text-ink shadow-2xl backdrop:bg-primary/40 sm:p-7"
      >
        {confirming ? (
          <>
            <h2 id="apply-confirm-title" className="text-lg font-bold text-primary">
              Close the application form?
            </h2>
            <p id="apply-confirm-text" className="mt-2 text-muted">
              Your answers and any attached files will be cleared. This cannot be undone.
            </p>
            <div className="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
              <button
                type="button"
                onClick={finish}
                className="inline-flex items-center justify-center rounded-md border border-red-700 px-5 py-3 text-sm font-semibold text-red-700 transition-colors duration-200 ease-out hover:bg-red-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-red-700"
              >
                Discard and close
              </button>
              <button ref={keepEditingButton} type="button" onClick={keepEditing} className={buttonClass('primary')}>
                Keep editing
              </button>
            </div>
          </>
        ) : null}
      </dialog>
    </ApplyContext.Provider>
  )
}

export function ApplyButton({ className }: { className?: string }) {
  const show = useContext(ApplyContext)
  if (!show) {
    return null
  }
  return (
    <button type="button" onClick={show} aria-haspopup="dialog" className={buttonClass('accent', className)}>
      Apply Now
    </button>
  )
}
