export function list<T>(value: readonly T[] | null | undefined): readonly T[] {
  return value ?? []
}
