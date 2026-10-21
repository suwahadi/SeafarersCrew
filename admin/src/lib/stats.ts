export function formatStat(stat: { number: number; suffix: string; label: string }): string {
  const grouped = String(stat.number).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  return `${grouped}${stat.suffix} ${stat.label}`.trim()
}
