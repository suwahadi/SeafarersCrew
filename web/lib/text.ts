export function firstNonEmpty(...values: readonly (string | null | undefined)[]): string {
  for (const value of values) {
    if (value !== null && value !== undefined && value.trim() !== '') {
      return value
    }
  }
  return ''
}
