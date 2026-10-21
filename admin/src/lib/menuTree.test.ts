import { describe, expect, it } from 'vitest'
import {
  appendNode,
  canIndent,
  canOutdent,
  fromServer,
  indentNode,
  makeNode,
  mapServerErrors,
  MENU_MAX_CHILDREN,
  MENU_MAX_ITEMS,
  moveNodeBy,
  moveNodeOnto,
  outdentNode,
  removeNode,
  toPayload,
  validateTree,
  type MenuNode,
} from './menuTree'

const show = (tree: MenuNode[]) => tree.map((n) => (n.children.length ? `${n.label}[${n.children.map((c) => c.label).join(',')}]` : n.label))

function build(...names: string[]): MenuNode[] {
  return names.map((n) => makeNode({ label: n, url: `/${n.toLowerCase()}` }))
}

function keyOf(tree: MenuNode[], index: number): string {
  const node = tree[index]
  if (!node) throw new Error(`no node at ${index}`)
  return node.key
}

describe('menu tree operations', () => {
  it('moves an item up and down within its level and stops at the ends', () => {
    const tree = build('A', 'B', 'C')
    expect(show(moveNodeBy(tree, keyOf(tree, 1), -1))).toEqual(['B', 'A', 'C'])
    expect(show(moveNodeBy(tree, keyOf(tree, 1), 1))).toEqual(['A', 'C', 'B'])
    expect(show(moveNodeBy(tree, keyOf(tree, 0), -1))).toEqual(['A', 'B', 'C'])
  })

  it('indents an item under the previous item and outdents it right after its parent', () => {
    const tree = build('A', 'B', 'C')
    const indented = indentNode(tree, keyOf(tree, 1))
    expect(show(indented)).toEqual(['A[B]', 'C'])
    const child = indented[0]?.children[0]
    expect(show(outdentNode(indented, child?.key ?? ''))).toEqual(['A', 'B', 'C'])
  })

  it('does not indent the first item', () => {
    const tree = build('A', 'B')
    expect(canIndent(tree, keyOf(tree, 0))).toBe(false)
    expect(show(indentNode(tree, keyOf(tree, 0)))).toEqual(['A', 'B'])
  })

  it('does not indent an item that already has children, so there is never a third level', () => {
    const parentWithChild = makeNode({ label: 'P' })
    parentWithChild.children = [makeNode({ label: 'X' })]
    const flat = [makeNode({ label: 'Q' }), parentWithChild]
    expect(canIndent(flat, parentWithChild.key)).toBe(false)
    expect(show(indentNode(flat, parentWithChild.key))).toEqual(['Q', 'P[X]'])
  })

  it('limits children per item and top-level items', () => {
    const prev = makeNode({ label: 'P' })
    prev.children = Array.from({ length: MENU_MAX_CHILDREN }, (_, i) => makeNode({ label: `c${i}` }))
    const next = makeNode({ label: 'N' })
    expect(canIndent([prev, next], next.key)).toBe(false)

    const parent = makeNode({ label: 'P2' })
    const child = makeNode({ label: 'c' })
    parent.children = [child]
    const full = [parent, ...Array.from({ length: MENU_MAX_ITEMS - 1 }, (_, i) => makeNode({ label: `t${i}` }))]
    expect(full).toHaveLength(MENU_MAX_ITEMS)
    expect(canOutdent(full, child.key)).toBe(false)
    expect(appendNode(full, makeNode({ label: 'extra' }))).toHaveLength(MENU_MAX_ITEMS)
  })

  it('reorders by dropping onto a sibling and ignores drops across levels', () => {
    const flat = build('A', 'B', 'C')
    expect(show(moveNodeOnto(flat, keyOf(flat, 0), keyOf(flat, 2)))).toEqual(['B', 'C', 'A'])
    const nested = indentNode(flat, keyOf(flat, 1))
    const child = nested[0]?.children[0]
    expect(show(moveNodeOnto(nested, keyOf(nested, 1), child?.key ?? ''))).toEqual(['A[B]', 'C'])
  })

  it('removes an item together with its children', () => {
    const flat = build('A', 'B', 'C')
    const nested = indentNode(flat, keyOf(flat, 1))
    expect(show(removeNode(nested, keyOf(nested, 0)))).toEqual(['C'])
    const child = nested[0]?.children[0]
    expect(show(removeNode(nested, child?.key ?? ''))).toEqual(['A', 'C'])
  })
})

describe('menu payload', () => {
  it('sends page_id for page items and url for custom links, with children', () => {
    const page = makeNode({ label: ' About ', pageId: 'p1' })
    const link = makeNode({ label: 'Docs', url: ' https://docs.example.com ', openInNewTab: true })
    page.children = [makeNode({ label: 'Team', pageId: 'p2' })]
    expect(toPayload([page, link])).toEqual({
      items: [
        { label: 'About', page_id: 'p1', open_in_new_tab: false, children: [{ label: 'Team', page_id: 'p2', open_in_new_tab: false }] },
        { label: 'Docs', url: 'https://docs.example.com', open_in_new_tab: true, children: [] },
      ],
    })
  })

  it('builds the tree from the server response', () => {
    const tree = fromServer([
      {
        id: '1',
        label: 'About',
        page_id: 'p1',
        url: '',
        open_in_new_tab: false,
        page: { title: 'About us', path: '/about', status: 'draft' },
        children: [{ id: '2', label: 'Team', page_id: null, url: '/team', open_in_new_tab: true }],
      },
    ])
    expect(tree[0]?.page?.status).toBe('draft')
    expect(tree[0]?.children[0]).toMatchObject({ url: '/team', openInNewTab: true })
  })
})

describe('menu errors', () => {
  it('maps server paths to the right row', () => {
    const tree = build('A', 'B')
    const parent = tree[0]
    if (!parent) throw new Error('missing')
    const child = makeNode({ label: 'Child', url: 'x' })
    parent.children = [child]
    const mapped = mapServerErrors(tree, { 'items.0.children.0.url': 'Bad link', 'items.1.label': 'Too long', other: 'Something' })
    expect(mapped.byKey[child.key]).toEqual({ url: 'Bad link' })
    expect(mapped.byKey[keyOf(tree, 1)]).toEqual({ label: 'Too long' })
    expect(mapped.general).toEqual(['Something'])
  })

  it('validates labels and links before saving', () => {
    const tree = [makeNode({ label: '', url: '/ok' }), makeNode({ label: 'Bad', url: 'nope' }), makeNode({ label: 'Page', pageId: 'p' })]
    const errors = validateTree(tree, (u) => u.startsWith('/'))
    expect(errors[keyOf(tree, 0)]).toEqual({ label: 'Enter a label.' })
    expect(errors[keyOf(tree, 1)]).toEqual({ url: 'Enter a valid link.' })
    expect(errors[keyOf(tree, 2)]).toBeUndefined()
  })
})
