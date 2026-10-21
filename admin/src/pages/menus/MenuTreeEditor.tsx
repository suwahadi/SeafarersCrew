import { useDragSort } from '@/hooks/useDragSort'
import { Badge } from '@/components/ui/Layout'
import { Button } from '@/components/ui/Button'
import { canIndent, canMoveDown, canMoveUp, canOutdent, indentNode, moveNodeBy, moveNodeOnto, outdentNode, removeNode, updateNode, type MenuNode, type NodeErrors } from '@/lib/menuTree'

interface MenuTreeEditorProps {
  tree: MenuNode[]
  errors: NodeErrors
  onChange: (tree: MenuNode[]) => void
}

export function MenuTreeEditor({ tree, errors, onChange }: MenuTreeEditorProps) {
  const drag = useDragSort((from, to) => onChange(moveNodeOnto(tree, from, to)))

  if (tree.length === 0) {
    return <p className="rounded-lg border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500">This menu is empty. Add a page or a custom link below.</p>
  }

  const renderNode = (node: MenuNode, nested: boolean) => {
    const err = errors[node.key] ?? {}
    const name = node.label.trim() || 'item'
    const draft = node.page?.status === 'draft'
    return (
      <li key={node.key} {...drag.targetProps(node.key)} className={`transition-opacity duration-150 ease-out motion-reduce:transition-none ${drag.dragging === node.key ? 'opacity-50' : 'opacity-100'}`}>
        <div className="rounded-md bg-white p-3 ring-1 ring-slate-200">
          <div className="flex flex-wrap items-start gap-3">
            <span {...drag.handleProps(node.key)} title="Drag to reorder" aria-hidden className="mt-1.5 cursor-grab select-none px-1 text-slate-400">
              ⋮⋮
            </span>
            <div className="min-w-48 flex-1 space-y-1">
              <label className="block text-xs font-medium text-slate-600">
                Label
                <input
                  value={node.label}
                  maxLength={80}
                  aria-invalid={err.label ? true : undefined}
                  onChange={(e) => onChange(updateNode(tree, node.key, { label: e.target.value }))}
                  className="mt-0.5 block w-full rounded-md border-0 px-2 py-1 text-sm text-slate-900 ring-1 ring-inset ring-slate-300 aria-[invalid=true]:ring-red-500"
                />
              </label>
              {err.label ? (
                <p role="alert" className="text-xs text-red-600">
                  {err.label}
                </p>
              ) : null}
            </div>
            <div className="min-w-48 flex-1 space-y-1">
              {node.pageId ? (
                <div className="pt-4 text-sm">
                  <span className="mr-2 text-slate-700">{node.page?.title ?? 'Page'}</span>
                  <span className="text-xs text-slate-500">{node.page?.path}</span>{' '}
                  {draft ? <Badge tone="yellow">Draft</Badge> : <Badge tone="green">Published</Badge>}
                </div>
              ) : (
                <label className="block text-xs font-medium text-slate-600">
                  Link
                  <input
                    value={node.url}
                    aria-invalid={err.url ? true : undefined}
                    onChange={(e) => onChange(updateNode(tree, node.key, { url: e.target.value }))}
                    className="mt-0.5 block w-full rounded-md border-0 px-2 py-1 text-sm text-slate-900 ring-1 ring-inset ring-slate-300 aria-[invalid=true]:ring-red-500"
                  />
                </label>
              )}
              {err.url || err.page_id ? (
                <p role="alert" className="text-xs text-red-600">
                  {err.url ?? err.page_id}
                </p>
              ) : null}
              {draft ? <p className="text-xs text-yellow-800">This page is a draft and stays hidden on the site until it is published.</p> : null}
            </div>
            <label className="flex items-center gap-2 pt-5 text-xs text-slate-700">
              <input type="checkbox" checked={node.openInNewTab} onChange={(e) => onChange(updateNode(tree, node.key, { openInNewTab: e.target.checked }))} className="h-4 w-4 rounded border-slate-300" />
              New tab
            </label>
          </div>
          <div className="mt-2 flex flex-wrap gap-1">
            <Button size="sm" variant="ghost" aria-label={`Move ${name} up`} disabled={!canMoveUp(tree, node.key)} onClick={() => onChange(moveNodeBy(tree, node.key, -1))}>
              ↑
            </Button>
            <Button size="sm" variant="ghost" aria-label={`Move ${name} down`} disabled={!canMoveDown(tree, node.key)} onClick={() => onChange(moveNodeBy(tree, node.key, 1))}>
              ↓
            </Button>
            {nested ? (
              <Button size="sm" variant="ghost" aria-label={`Outdent ${name}`} disabled={!canOutdent(tree, node.key)} onClick={() => onChange(outdentNode(tree, node.key))}>
                ← Outdent
              </Button>
            ) : (
              <Button
                size="sm"
                variant="ghost"
                aria-label={`Indent ${name}`}
                title={node.children.length > 0 ? 'Items with sub-items cannot be nested further.' : undefined}
                disabled={!canIndent(tree, node.key)}
                onClick={() => onChange(indentNode(tree, node.key))}
              >
                Indent →
              </Button>
            )}
            <Button size="sm" variant="ghost" aria-label={`Remove ${name}`} onClick={() => onChange(removeNode(tree, node.key))}>
              Remove
            </Button>
          </div>
        </div>
        {node.children.length > 0 ? (
          <ol aria-label={`Sub-items of ${name}`} className="ml-8 mt-2 space-y-2 border-l-2 border-slate-200 pl-3">
            {node.children.map((c) => renderNode(c, true))}
          </ol>
        ) : null}
      </li>
    )
  }

  return (
    <ol aria-label="Menu items" className="space-y-2">
      {tree.map((n) => renderNode(n, false))}
    </ol>
  )
}
