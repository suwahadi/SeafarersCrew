export function isValidIMO(value: string): boolean {
  if (!/^\d{7}$/.test(value)) return false
  const digits = value.split('').map(Number)
  let sum = 0
  for (let i = 0; i < 6; i++) sum += (digits[i] ?? 0) * (7 - i)
  return sum % 10 === digits[6]
}
