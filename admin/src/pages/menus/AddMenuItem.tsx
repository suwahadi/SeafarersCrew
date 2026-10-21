import { useState } from 'react'
import type { PageSummary } from '@/api/cms'
import { Button } from '@/components/ui/Button'
import { SelectField, TextField } from '@/components/ui/Field'
import { Card } from '@/components/ui/Layout'
import { isValidLink, LINK_FORMAT_MESSAGE } from '@/lib/links'
import { makeNode, type MenuNode } from '@/lib/menuTree'

interface AddMenuItemProps {
  pages: PageSummary[]
  pagesLoading: boolean
  disabled: boolean
  onAdd: (node: MenuNode) => void
}

export function AddMenuItem({ pages, pagesLoading, disabled, onAdd }: AddMenuItemProps) {
  const [pageId, setPageId] = useState('')
  const [label, setLabel] = useState('')
  const [url, setUrl] = useState('')
  const [newTab, setNewTab] = useState(false)
  const [errors, setErrors] = useState<{ label?: string; url?: string }>({})
  const selected = pages.find((p) => p.id === pageId)

  const addPage = () => {
    if (!selected) return
    onAdd(makeNode({ label: selected.title, pageId: selected.id, page: { title: selected.title, path: selected.path, status: selected.status } }))
    setPageId('')
  }

  const addLink = () => {
    const next: { label?: string; url?: string } = {}
    if (label.trim() === '') next.label = 'Enter a label.'
    else if (label.trim().length > 80) next.label = 'The label may not exceed 80 characters.'
    if (!isValidLink(url)) next.url = LINK_FORMAT_MESSAGE
    setErrors(next)
    if (next.label || next.url) return
    onAdd(makeNode({ label: label.trim(), url: url.trim(), openInNewTab: newTab }))
    setLabel('')
    setUrl('')
    setNewTab(false)
  }

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <Card title="Add a page">
        <div className="space-y-3">
          <SelectField
            label="Page"
            value={pageId}
            onChange={(e) => setPageId(e.target.value)}
            placeholder={pagesLoading ? 'Loading pages…' : 'Choose a page'}
            options={pages.map((p) => ({ value: p.id, label: `${p.title} (${p.path})${p.status === 'draft' ? ' — draft' : ''}${p.is_system ? ' — system page' : ''}` }))}
          />
          {selected?.status === 'draft' ? <p className="text-xs text-yellow-800">This page is a draft. The menu item is skipped on the site until the page is published.</p> : null}
          <Button variant="secondary" disabled={!selected || disabled} onClick={addPage}>
            Add page
          </Button>
        </div>
      </Card>
      <Card title="Add a custom link">
        <div className="space-y-3">
          <TextField label="Label" value={label} maxLength={80} onChange={(e) => setLabel(e.target.value)} error={errors.label} />
          <TextField label="URL" value={url} hint="/path, https://…, mailto: or tel:" onChange={(e) => setUrl(e.target.value)} error={errors.url} />
          <label className="flex items-center gap-2 text-sm text-slate-700">
            <input type="checkbox" checked={newTab} onChange={(e) => setNewTab(e.target.checked)} className="h-4 w-4 rounded border-slate-300" />
            Open in a new tab
          </label>
          <Button variant="secondary" disabled={disabled} onClick={addLink}>
            Add link
          </Button>
        </div>
      </Card>
    </div>
  )
}
