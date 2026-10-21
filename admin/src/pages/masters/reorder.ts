export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
  if (from === to || from < 0 || to < 0 || from >= items.length || to >= items.length) return [...items]
  const out = [...items]
  const [moved] = out.splice(from, 1)
  if (moved === undefined) return [...items]
  out.splice(to, 0, moved)
  return out
}
