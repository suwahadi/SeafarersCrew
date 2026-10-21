import Image from '@tiptap/extension-image'
import { Table, TableCell, TableHeader, TableRow } from '@tiptap/extension-table'
import { EditorContent, useEditor, type Editor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { useEffect, useId, useState, type ReactNode } from 'react'
import type { VariableGroup } from '@/api/templates'
import { PageBreak } from './pageBreak'
import { VariablePicker } from './VariablePicker'

interface RichTextEditorProps {
  label: string
  value: string
  onChange: (html: string) => void
  groups?: VariableGroup[]
  error?: string | undefined
  hint?: string | undefined
  minHeight?: string
  withLayout?: boolean
}

const SAFE_URL = /^(https?:\/\/|mailto:|tel:)/i

export function normalizeHtml(html: string): string {
  return html === '<p></p>' ? '' : html
}

export function isSafeUrl(url: string): boolean {
  return SAFE_URL.test(url.trim())
}

function ToolButton({ label, active, disabled, onClick, children }: { label: string; active?: boolean; disabled?: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      aria-pressed={active}
      disabled={disabled}
      onMouseDown={(e) => e.preventDefault()}
      onClick={onClick}
      className={`h-8 min-w-8 rounded px-2 text-sm transition-colors duration-150 disabled:opacity-40 ${active ? 'bg-navy-100 text-navy-900' : 'text-slate-700 hover:bg-slate-100'}`}
    >
      {children}
    </button>
  )
}

type UrlMode = 'link' | 'image' | null

function UrlBar({ mode, onApply, onCancel }: { mode: Exclude<UrlMode, null>; onApply: (url: string) => void; onCancel: () => void }) {
  const [url, setUrl] = useState('')
  const [problem, setProblem] = useState<string | null>(null)
  const apply = () => {
    if (url.trim() !== '' && !isSafeUrl(url)) {
      setProblem('Use an http, https, mailto or tel address.')
      return
    }
    onApply(url.trim())
  }
  return (
    <div className="flex flex-wrap items-center gap-2 border-t border-slate-200 bg-slate-50 px-2 py-1">
      <label className="text-xs text-slate-600" htmlFor={`rte-url-${mode}`}>
        {mode === 'link' ? 'Link address' : 'Image address'}
      </label>
      <input
        id={`rte-url-${mode}`}
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault()
            apply()
          }
          if (e.key === 'Escape') onCancel()
        }}
        placeholder="https://"
        className="h-8 min-w-64 flex-1 rounded-md border-0 text-sm ring-1 ring-inset ring-slate-300"
      />
      <button type="button" onClick={apply} className="h-8 rounded-md bg-navy-800 px-3 text-sm text-white hover:bg-navy-700">
        {mode === 'link' && url.trim() === '' ? 'Remove link' : 'Apply'}
      </button>
      <button type="button" onClick={onCancel} className="h-8 rounded-md px-3 text-sm text-slate-700 hover:bg-slate-100">
        Cancel
      </button>
      {problem ? (
        <p role="alert" className="w-full text-xs text-red-600">
          {problem}
        </p>
      ) : null}
    </div>
  )
}

function Toolbar({ editor, groups, withLayout }: { editor: Editor; groups: VariableGroup[]; withLayout: boolean }) {
  const [urlMode, setUrlMode] = useState<UrlMode>(null)
  const chain = () => editor.chain().focus()

  const applyUrl = (url: string) => {
    if (urlMode === 'link') {
      if (url === '') chain().extendMarkRange('link').unsetLink().run()
      else chain().extendMarkRange('link').setLink({ href: url }).run()
    } else if (urlMode === 'image' && url !== '') {
      chain().setImage({ src: url }).run()
    }
    setUrlMode(null)
  }

  return (
    <div className="rounded-t-md border-b border-slate-200 bg-white">
      <div role="toolbar" aria-label="Formatting" className="flex flex-wrap items-center gap-0.5 p-1">
        <ToolButton label="Heading 2" active={editor.isActive('heading', { level: 2 })} onClick={() => chain().toggleHeading({ level: 2 }).run()}>
          H2
        </ToolButton>
        <ToolButton label="Heading 3" active={editor.isActive('heading', { level: 3 })} onClick={() => chain().toggleHeading({ level: 3 }).run()}>
          H3
        </ToolButton>
        <ToolButton label="Bold" active={editor.isActive('bold')} onClick={() => chain().toggleBold().run()}>
          <strong>B</strong>
        </ToolButton>
        <ToolButton label="Italic" active={editor.isActive('italic')} onClick={() => chain().toggleItalic().run()}>
          <em>I</em>
        </ToolButton>
        <ToolButton label="Underline" active={editor.isActive('underline')} onClick={() => chain().toggleUnderline().run()}>
          <span className="underline">U</span>
        </ToolButton>
        <ToolButton label="Bullet list" active={editor.isActive('bulletList')} onClick={() => chain().toggleBulletList().run()}>
          • List
        </ToolButton>
        <ToolButton label="Numbered list" active={editor.isActive('orderedList')} onClick={() => chain().toggleOrderedList().run()}>
          1. List
        </ToolButton>
        <ToolButton label="Link" active={editor.isActive('link')} onClick={() => setUrlMode('link')}>
          Link
        </ToolButton>
        {withLayout ? (
          <>
            <ToolButton label="Insert image" onClick={() => setUrlMode('image')}>
              Image
            </ToolButton>
            <ToolButton label="Insert table" onClick={() => chain().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()}>
              Table
            </ToolButton>
            <ToolButton label="Add table row" disabled={!editor.isActive('table')} onClick={() => chain().addRowAfter().run()}>
              +Row
            </ToolButton>
            <ToolButton label="Add table column" disabled={!editor.isActive('table')} onClick={() => chain().addColumnAfter().run()}>
              +Col
            </ToolButton>
            <ToolButton label="Delete table" disabled={!editor.isActive('table')} onClick={() => chain().deleteTable().run()}>
              ×Table
            </ToolButton>
            <ToolButton label="Insert page break" onClick={() => chain().insertContent({ type: 'pageBreak' }).run()}>
              Page break
            </ToolButton>
          </>
        ) : null}
        <ToolButton label="Undo" disabled={!editor.can().undo()} onClick={() => chain().undo().run()}>
          ↶
        </ToolButton>
        <ToolButton label="Redo" disabled={!editor.can().redo()} onClick={() => chain().redo().run()}>
          ↷
        </ToolButton>
        {groups.length > 0 ? (
          <span className="ml-auto">
            <VariablePicker groups={groups} onInsert={(text) => chain().insertContent(text).run()} />
          </span>
        ) : null}
      </div>
      {urlMode ? <UrlBar key={urlMode} mode={urlMode} onApply={applyUrl} onCancel={() => setUrlMode(null)} /> : null}
    </div>
  )
}

export function RichTextEditor({ label, value, onChange, groups = [], error, hint, minHeight = '12rem', withLayout = true }: RichTextEditorProps) {
  const id = useId()
  const editor = useEditor({
    extensions: [
      StarterKit.configure({ heading: { levels: [1, 2, 3] }, link: { openOnClick: false, autolink: false } }),
      Image,
      Table.configure({ resizable: false }),
      TableRow,
      TableHeader,
      TableCell,
      PageBreak,
    ],
    content: value,
    shouldRerenderOnTransaction: true,
    editorProps: {
      attributes: { id, role: 'textbox', 'aria-multiline': 'true', 'aria-label': label, class: 'rte-content', style: `--rte-min-height:${minHeight}` },
    },
    onUpdate: ({ editor: e }) => onChange(normalizeHtml(e.getHTML())),
  })

  useEffect(() => {
    if (normalizeHtml(editor.getHTML()) !== value) editor.commands.setContent(value, { emitUpdate: false })
  }, [editor, value])

  return (
    <div className="space-y-1">
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <div className={`rounded-md bg-white ring-1 ring-inset ${error ? 'ring-red-500' : 'ring-slate-300'} focus-within:ring-2 focus-within:ring-navy-500`}>
        <Toolbar editor={editor} groups={groups} withLayout={withLayout} />
        <EditorContent editor={editor} />
      </div>
      {error ? (
        <p role="alert" className="text-xs text-red-600">
          {error}
        </p>
      ) : hint ? (
        <p className="text-xs text-slate-500">{hint}</p>
      ) : null}
    </div>
  )
}
