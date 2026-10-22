import { DragHandle } from '@/components/ui/DragHandle'
import { IconButton } from '@/components/ui/IconButton'
import { inputClass } from '@/components/ui/Field'
import { Badge } from '@/components/ui/Layout'
import { useSortable } from '@/hooks/useSortable'
import { canIndent, canOutdent, indentNode, moveNodeOnto, outdentNode, removeNode, updateNode, type MenuNode, type NodeErrors } from '@/lib/menuTree'

interface MenuTreeEditorProps {
  tree: MenuNode[]
  errors: NodeErrors
  onChange: (tree: MenuNode[]) => void
}

interface LevelProps extends MenuTreeEditorProps {
  nodes: MenuNode[]
  nested: boolean
  label: string
}

function MenuLevel({ nodes, nested, label, tree, errors, onChange }: LevelProps) {
  const sort = useSortable(
    nodes.map((n) => n.key),
    (from, to) => onChange(moveNodeOnto(tree, from, to)),
  )

  return (
    <ol aria-label={label} className={nested ? 'ml-3 mt-2 space-y-2 border-l-2 border-slate-200 pl-2 sm:ml-8 sm:pl-3' : 'space-y-2'}>
      {nodes.map((node) => {
        const err = errors[node.key] ?? {}
        const name = node.label.trim() || 'item'
        const draft = node.page?.status === 'draft'
        return (
          <li key={node.key} {...sort.rowProps(node.key)}>
            <div className="rounded-lg bg-white p-3 ring-1 ring-slate-200">
              <div className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-2 gap-y-3 sm:flex sm:flex-wrap sm:items-start sm:gap-3">
                <DragHandle label={name} className="row-span-3 mt-5 self-start sm:row-span-1" {...sort.handleProps(node.key)} />
                <div className="min-w-0 space-y-1 sm:min-w-48 sm:flex-1">
                  <label className="block text-xs font-medium text-slate-600">
                    Label
                    <input
                      value={node.label}
                      maxLength={80}
                      aria-invalid={err.label ? true : undefined}
                      onChange={(e) => onChange(updateNode(tree, node.key, { label: e.target.value }))}
                      className={`mt-1 font-normal ${inputClass}`}
                    />
                  </label>
                  {err.label ? (
                    <p role="alert" className="text-xs text-red-600">
                      {err.label}
                    </p>
                  ) : null}
                </div>
                <div className="min-w-0 space-y-1 sm:min-w-48 sm:flex-1">
                  {node.pageId ? (
                    <div>
                      <span className="block text-xs font-medium text-slate-600">Page</span>
                      <div className="mt-1 flex min-h-8 flex-wrap items-center gap-x-2 text-sm">
                        <span className="text-slate-800">{node.page?.title ?? 'Page'}</span>
                        <span className="break-all text-xs text-slate-500">{node.page?.path}</span>
                        {draft ? <Badge tone="yellow">Draft</Badge> : <Badge tone="green">Published</Badge>}
                      </div>
                    </div>
                  ) : (
                    <label className="block text-xs font-medium text-slate-600">
                      Link
                      <input
                        value={node.url}
                        aria-invalid={err.url ? true : undefined}
                        onChange={(e) => onChange(updateNode(tree, node.key, { url: e.target.value }))}
                        className={`mt-1 font-normal ${inputClass}`}
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
                <div className="flex flex-wrap items-center justify-between gap-2 sm:contents">
                  <label className="flex h-9 items-center gap-2 text-sm text-slate-700 sm:mt-5">
                    <input type="checkbox" checked={node.openInNewTab} onChange={(e) => onChange(updateNode(tree, node.key, { openInNewTab: e.target.checked }))} className="h-4 w-4 rounded border-slate-300" />
                    New tab
                  </label>
                  <div className="flex gap-1 sm:mt-5">
                    {nested ? (
                      <IconButton icon="outdent" label={`Outdent ${name}`} disabled={!canOutdent(tree, node.key)} onClick={() => onChange(outdentNode(tree, node.key))} />
                    ) : (
                      <IconButton
                        icon="indent"
                        label={`Indent ${name}`}
                        title={node.children.length > 0 ? 'Items with sub-items cannot be nested further.' : `Indent ${name}`}
                        disabled={!canIndent(tree, node.key)}
                        onClick={() => onChange(indentNode(tree, node.key))}
                      />
                    )}
                    <IconButton icon="trash" tone="danger" label={`Remove ${name}`} onClick={() => onChange(removeNode(tree, node.key))} />
                  </div>
                </div>
              </div>
            </div>
            {node.children.length > 0 ? <MenuLevel nodes={node.children} nested label={`Sub-items of ${name}`} tree={tree} errors={errors} onChange={onChange} /> : null}
          </li>
        )
      })}
    </ol>
  )
}

export function MenuTreeEditor({ tree, errors, onChange }: MenuTreeEditorProps) {
  if (tree.length === 0) {
    return <p className="rounded-lg border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500">This menu is empty. Add a page or a custom link below.</p>
  }
  return <MenuLevel nodes={tree} nested={false} label="Menu items" tree={tree} errors={errors} onChange={onChange} />
}
