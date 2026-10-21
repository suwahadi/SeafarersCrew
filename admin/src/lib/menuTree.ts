export const MENU_MAX_ITEMS = 30
export const MENU_MAX_CHILDREN = 20

export interface MenuNode {
  key: string
  label: string
  pageId: string | null
  url: string
  openInNewTab: boolean
  page: NodePage | null
  children: MenuNode[]
}

export interface NodePage {
  title: string
  path: string
  status: string
}

export interface ServerMenuItem {
  id: string
  label: string
  page_id: string | null
  url: string
  open_in_new_tab: boolean
  page?: { title: string; path: string; status: string } | null
  children?: ServerMenuItem[]
}

export interface MenuPayloadItem {
  label: string
  page_id?: string
  url?: string
  open_in_new_tab: boolean
  children?: MenuPayloadItem[]
}

let counter = 0

function newKey(): string {
  counter += 1
  return `n${counter}`
}

export function makeNode(init: { label: string; pageId?: string | null; url?: string; openInNewTab?: boolean; page?: NodePage | null }): MenuNode {
  return { key: newKey(), label: init.label, pageId: init.pageId ?? null, url: init.url ?? '', openInNewTab: init.openInNewTab ?? false, page: init.page ?? null, children: [] }
}

function fromServerItem(item: ServerMenuItem, withChildren: boolean): MenuNode {
  return {
    key: newKey(),
    label: item.label,
    pageId: item.page_id,
    url: item.page_id ? '' : item.url,
    openInNewTab: item.open_in_new_tab,
    page: item.page ? { title: item.page.title, path: item.page.path, status: item.page.status } : null,
    children: withChildren ? (item.children ?? []).map((c) => fromServerItem(c, false)) : [],
  }
}

export function fromServer(items: readonly ServerMenuItem[]): MenuNode[] {
  return items.map((i) => fromServerItem(i, true))
}

function nodePayload(node: MenuNode): MenuPayloadItem {
  const base: MenuPayloadItem = { label: node.label.trim(), open_in_new_tab: node.openInNewTab }
  if (node.pageId) base.page_id = node.pageId
  else base.url = node.url.trim()
  return base
}

export function toPayload(tree: readonly MenuNode[]): { items: MenuPayloadItem[] } {
  return { items: tree.map((n) => ({ ...nodePayload(n), children: n.children.map(nodePayload) })) }
}

export function payloadSignature(tree: readonly MenuNode[]): string {
  return JSON.stringify(toPayload(tree))
}

export function findNode(tree: readonly MenuNode[], key: string): { node: MenuNode; parent: MenuNode | null; index: number } | null {
  for (const [index, node] of tree.entries()) {
    if (node.key === key) return { node, parent: null, index }
    const childIndex = node.children.findIndex((c) => c.key === key)
    const child = node.children[childIndex]
    if (child) return { node: child, parent: node, index: childIndex }
  }
  return null
}

function relocate<T>(list: readonly T[], from: number, to: number): T[] {
  const out = [...list]
  const [moved] = out.splice(from, 1)
  if (moved === undefined) return out
  out.splice(to, 0, moved)
  return out
}

function mapSiblings(tree: readonly MenuNode[], key: string, fn: (siblings: MenuNode[], index: number) => MenuNode[]): MenuNode[] {
  const top = tree.findIndex((n) => n.key === key)
  if (top >= 0) return fn([...tree], top)
  return tree.map((n) => {
    const index = n.children.findIndex((c) => c.key === key)
    return index < 0 ? n : { ...n, children: fn([...n.children], index) }
  })
}

export function moveNodeBy(tree: readonly MenuNode[], key: string, delta: -1 | 1): MenuNode[] {
  return mapSiblings(tree, key, (siblings, index) => {
    const target = index + delta
    return target < 0 || target >= siblings.length ? siblings : relocate(siblings, index, target)
  })
}

export function moveNodeOnto(tree: readonly MenuNode[], fromKey: string, toKey: string): MenuNode[] {
  const from = findNode(tree, fromKey)
  const to = findNode(tree, toKey)
  if (!from || !to || from.parent?.key !== to.parent?.key) return [...tree]
  return mapSiblings(tree, fromKey, (siblings, index) => relocate(siblings, index, to.index))
}

export function canMoveUp(tree: readonly MenuNode[], key: string): boolean {
  const found = findNode(tree, key)
  return found !== null && found.index > 0
}

export function canMoveDown(tree: readonly MenuNode[], key: string): boolean {
  const found = findNode(tree, key)
  if (!found) return false
  const siblings = found.parent ? found.parent.children : tree
  return found.index < siblings.length - 1
}

export function canIndent(tree: readonly MenuNode[], key: string): boolean {
  const found = findNode(tree, key)
  if (found?.parent !== null || found.index === 0 || found.node.children.length > 0) return false
  const previous = tree[found.index - 1]
  return previous !== undefined && previous.children.length < MENU_MAX_CHILDREN
}

export function canOutdent(tree: readonly MenuNode[], key: string): boolean {
  const found = findNode(tree, key)
  return found !== null && found.parent !== null && tree.length < MENU_MAX_ITEMS
}

export function indentNode(tree: readonly MenuNode[], key: string): MenuNode[] {
  const found = findNode(tree, key)
  if (!found || !canIndent(tree, key)) return [...tree]
  return tree.filter((n) => n.key !== key).map((n, i) => (i === found.index - 1 ? { ...n, children: [...n.children, found.node] } : n))
}

export function outdentNode(tree: readonly MenuNode[], key: string): MenuNode[] {
  const found = findNode(tree, key)
  if (!found?.parent || !canOutdent(tree, key)) return [...tree]
  const parentKey = found.parent.key
  const out: MenuNode[] = []
  for (const n of tree) {
    if (n.key !== parentKey) {
      out.push(n)
      continue
    }
    out.push({ ...n, children: n.children.filter((c) => c.key !== key) })
    out.push(found.node)
  }
  return out
}

export function removeNode(tree: readonly MenuNode[], key: string): MenuNode[] {
  return tree.filter((n) => n.key !== key).map((n) => ({ ...n, children: n.children.filter((c) => c.key !== key) }))
}

export function updateNode(tree: readonly MenuNode[], key: string, patch: Partial<Pick<MenuNode, 'label' | 'url' | 'openInNewTab'>>): MenuNode[] {
  return tree.map((n) => (n.key === key ? { ...n, ...patch } : { ...n, children: n.children.map((c) => (c.key === key ? { ...c, ...patch } : c)) }))
}

export function appendNode(tree: readonly MenuNode[], node: MenuNode): MenuNode[] {
  return tree.length >= MENU_MAX_ITEMS ? [...tree] : [...tree, node]
}

export type NodeErrors = Record<string, Record<string, string>>

export interface MenuErrors {
  byKey: NodeErrors
  general: string[]
}

export function mapServerErrors(tree: readonly MenuNode[], fields: Record<string, string>): MenuErrors {
  const out: MenuErrors = { byKey: {}, general: [] }
  const place = (key: string | undefined, field: string, message: string) => {
    if (!key) {
      out.general.push(message)
      return
    }
    out.byKey[key] = { ...out.byKey[key], [field]: message }
  }
  for (const [path, message] of Object.entries(fields)) {
    const child = /^items\.(\d+)\.children\.(\d+)\.(\w+)$/.exec(path)
    if (child) {
      place(tree[Number(child[1])]?.children[Number(child[2])]?.key, child[3] ?? '', message)
      continue
    }
    const top = /^items\.(\d+)\.(\w+)$/.exec(path)
    if (top) {
      place(tree[Number(top[1])]?.key, top[2] ?? '', message)
      continue
    }
    out.general.push(message)
  }
  return out
}

export function validateTree(tree: readonly MenuNode[], isValidLink: (url: string) => boolean): NodeErrors {
  const errors: NodeErrors = {}
  const check = (n: MenuNode) => {
    const e: Record<string, string> = {}
    if (n.label.trim() === '') e.label = 'Enter a label.'
    else if (n.label.trim().length > 80) e.label = 'The label may not exceed 80 characters.'
    if (!n.pageId && !isValidLink(n.url)) e.url = 'Enter a valid link.'
    if (Object.keys(e).length > 0) errors[n.key] = e
  }
  for (const n of tree) {
    check(n)
    n.children.forEach(check)
  }
  return errors
}
