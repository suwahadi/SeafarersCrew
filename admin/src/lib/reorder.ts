export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
  if (from === to || from < 0 || to < 0 || from >= items.length || to >= items.length) return [...items]
  const out = [...items]
  const [moved] = out.splice(from, 1)
  if (moved === undefined) return [...items]
  out.splice(to, 0, moved)
  return out
}

export function moveById<T extends { id: string }>(items: readonly T[], fromId: string, toId: string): T[] {
  return moveItem(
    items,
    items.findIndex((i) => i.id === fromId),
    items.findIndex((i) => i.id === toId),
  )
}

export function moveByIdWithinSubset<T extends { id: string }>(all: readonly T[], visibleIds: ReadonlySet<string>, fromId: string, toId: string): T[] {
  const visible = all.filter((i) => visibleIds.has(i.id))
  const reordered = moveById(visible, fromId, toId)
  let next = 0
  return all.map((item) => {
    if (!visibleIds.has(item.id)) return item
    const replacement = reordered[next]
    next += 1
    return replacement ?? item
  })
}

export function stepById<T extends { id: string }>(all: readonly T[], visibleIds: ReadonlySet<string>, id: string, delta: -1 | 1): T[] {
  const visible = all.filter((i) => visibleIds.has(i.id))
  const index = visible.findIndex((i) => i.id === id)
  const target = visible[index + delta]
  if (index < 0 || !target) return [...all]
  return moveByIdWithinSubset(all, visibleIds, id, target.id)
}
