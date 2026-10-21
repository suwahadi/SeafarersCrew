export function Spinner({ size = 'md', label = 'Loading' }: { size?: 'sm' | 'md' | 'lg'; label?: string }) {
  const dim = size === 'sm' ? 'h-4 w-4' : size === 'lg' ? 'h-8 w-8' : 'h-5 w-5'
  return (
    <span role="status" aria-label={label} className={`inline-block ${dim} animate-spin rounded-full border-2 border-current border-r-transparent`} />
  )
}
