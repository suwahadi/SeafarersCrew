import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { ApiError, errorMessage } from '@/api/client'
import { cmsKey, galleryApi } from '@/api/cms'
import { IMAGE_ACCEPT } from '@/api/cmsMedia'
import { filesApi } from '@/api/files'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { buildBulkRequests, humanizeFileName, runWithConcurrency, UPLOAD_CONCURRENCY } from '@/lib/bulkUpload'

type EntryStatus = 'uploading' | 'ready' | 'failed' | 'saved'

interface Entry {
  key: string
  file: File
  status: EntryStatus
  fileId: string
  altText: string
  caption: string
  error: string | null
}

let entryCounter = 0

export function GalleryBulkUpload({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <Modal open={open} title="Bulk upload photos" onClose={onClose} size="xl">
      {open ? <BulkUploader onDone={onClose} /> : null}
    </Modal>
  )
}

export function BulkUploader({ onDone }: { onDone: () => void }) {
  const qc = useQueryClient()
  const toast = useToast()
  const inputId = useId()
  const [entries, setEntries] = useState<Entry[]>([])
  const [showOnHome, setShowOnHome] = useState(false)
  const [isActive, setIsActive] = useState(true)
  const [saveError, setSaveError] = useState<string | null>(null)

  const patch = (key: string, change: Partial<Entry>) => setEntries((cur) => cur.map((e) => (e.key === key ? { ...e, ...change } : e)))

  const uploadOne = async (entry: Entry): Promise<void> => {
    patch(entry.key, { status: 'uploading', error: null })
    try {
      const stored = await filesApi.upload(entry.file, 'cms_image')
      patch(entry.key, { status: 'ready', fileId: stored.id })
    } catch (err) {
      patch(entry.key, { status: 'failed', error: errorMessage(err) })
    }
  }

  const addFiles = (files: File[]) => {
    const fresh = files.map<Entry>((file) => {
      entryCounter += 1
      return { key: `f${entryCounter}`, file, status: 'uploading', fileId: '', altText: humanizeFileName(file.name), caption: '', error: null }
    })
    setEntries((cur) => [...cur, ...fresh])
    void runWithConcurrency(fresh, UPLOAD_CONCURRENCY, uploadOne)
  }

  const ready = entries.filter((e) => e.status === 'ready')
  const uploading = entries.filter((e) => e.status === 'uploading').length
  const finished = entries.filter((e) => e.status !== 'uploading').length

  const save = useMutation({
    mutationFn: async () => {
      const requests = buildBulkRequests(
        ready.map((e) => ({ fileId: e.fileId, altText: e.altText, caption: e.caption })),
        { showOnHome, isActive },
      )
      let offset = 0
      for (const request of requests) {
        const group = ready.slice(offset, offset + request.items.length)
        try {
          await galleryApi.bulk(request)
        } catch (err) {
          if (err instanceof ApiError) {
            for (const [path, message] of Object.entries(err.fields)) {
              const match = /^items\.(\d+)\./.exec(path)
              const target = match ? group[Number(match[1])] : undefined
              if (target) patch(target.key, { error: message })
            }
          }
          throw err
        }
        for (const e of group) patch(e.key, { status: 'saved' })
        offset += request.items.length
      }
      return requests.length
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: cmsKey('gallery') })
      toast.success(`${ready.length} ${ready.length === 1 ? 'photo' : 'photos'} added.`)
      onDone()
    },
    onError: async (err) => {
      setSaveError(errorMessage(err))
      await qc.invalidateQueries({ queryKey: cmsKey('gallery') })
    },
  })

  const remaining = entries.filter((e) => e.status !== 'saved')

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-4">
        <label htmlFor={inputId} className="inline-flex h-9 cursor-pointer items-center rounded-md bg-white px-4 text-sm font-medium ring-1 ring-inset ring-slate-300 hover:bg-slate-50 focus-within:ring-2 focus-within:ring-navy-500">
          <input
            id={inputId}
            type="file"
            multiple
            accept={IMAGE_ACCEPT}
            className="sr-only"
            onChange={(e) => {
              addFiles(Array.from(e.target.files ?? []))
              e.target.value = ''
            }}
          />
          Choose images
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-700">
          <input type="checkbox" checked={showOnHome} onChange={(e) => setShowOnHome(e.target.checked)} className="h-4 w-4 rounded border-slate-300" />
          Show all on home page
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-700">
          <input type="checkbox" checked={isActive} onChange={(e) => setIsActive(e.target.checked)} className="h-4 w-4 rounded border-slate-300" />
          Active
        </label>
        {entries.length > 0 ? (
          <p className="text-sm text-slate-600" aria-live="polite">
            {finished} of {entries.length} processed
          </p>
        ) : null}
      </div>
      {saveError ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
          {saveError}
        </p>
      ) : null}
      {remaining.length === 0 ? (
        <p className="rounded-lg border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500">Choose one or more images. Each is uploaded as soon as you pick it.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="text-xs uppercase text-slate-500">
              <tr>
                <th className="py-2 pr-3 font-medium">File</th>
                <th className="py-2 pr-3 font-medium">Alt text</th>
                <th className="py-2 pr-3 font-medium">Caption</th>
                <th className="py-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {remaining.map((e) => (
                <tr key={e.key} className="align-top">
                  <td className="max-w-48 truncate py-2 pr-3 text-slate-700" title={e.file.name}>
                    {e.file.name}
                  </td>
                  <td className="py-2 pr-3">
                    <input
                      aria-label={`Alt text for ${e.file.name}`}
                      value={e.altText}
                      maxLength={200}
                      onChange={(ev) => patch(e.key, { altText: ev.target.value })}
                      className="w-full rounded-md border-0 px-2 py-1 text-sm ring-1 ring-inset ring-slate-300"
                    />
                  </td>
                  <td className="py-2 pr-3">
                    <input
                      aria-label={`Caption for ${e.file.name}`}
                      value={e.caption}
                      maxLength={300}
                      onChange={(ev) => patch(e.key, { caption: ev.target.value })}
                      className="w-full rounded-md border-0 px-2 py-1 text-sm ring-1 ring-inset ring-slate-300"
                    />
                  </td>
                  <td className="py-2">
                    {e.status === 'uploading' ? (
                      <span className="inline-flex items-center gap-2 text-slate-600">
                        <Spinner size="sm" label={`Uploading ${e.file.name}`} /> Uploading
                      </span>
                    ) : e.status === 'failed' ? (
                      <span className="space-y-1">
                        <span role="alert" className="block text-xs text-red-600">
                          {e.error}
                        </span>
                        <span className="flex gap-1">
                          <Button size="sm" variant="secondary" onClick={() => void uploadOne(e)}>
                            Retry
                          </Button>
                          <Button size="sm" variant="ghost" onClick={() => setEntries((cur) => cur.filter((x) => x.key !== e.key))}>
                            Remove
                          </Button>
                        </span>
                      </span>
                    ) : (
                      <span className="space-y-1">
                        <span className="block text-green-700">Ready</span>
                        {e.error ? (
                          <span role="alert" className="block text-xs text-red-600">
                            {e.error}
                          </span>
                        ) : null}
                        <Button size="sm" variant="ghost" onClick={() => setEntries((cur) => cur.filter((x) => x.key !== e.key))}>
                          Remove
                        </Button>
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <div className="flex justify-end gap-2 border-t border-slate-200 pt-4">
        <Button variant="secondary" onClick={onDone} disabled={save.isPending}>
          Cancel
        </Button>
        <Button
          onClick={() => {
            setSaveError(null)
            save.mutate()
          }}
          loading={save.isPending}
          disabled={ready.length === 0 || uploading > 0}
        >
          Add {ready.length} {ready.length === 1 ? 'photo' : 'photos'}
        </Button>
      </div>
    </div>
  )
}
