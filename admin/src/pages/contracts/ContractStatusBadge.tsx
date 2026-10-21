import { Badge, type Tone } from '@/components/ui/Layout'

const tones: Record<string, Tone> = {
  draft: 'neutral',
  active: 'blue',
  completed: 'green',
  terminated: 'red',
  planned: 'neutral',
  confirmed: 'blue',
  cancelled: 'red',
  sent: 'blue',
  accepted: 'green',
  rejected: 'red',
  expired: 'orange',
  queued: 'yellow',
  failed: 'red',
}

const labels: Record<string, string> = {
  draft: 'Draft',
  active: 'Active',
  completed: 'Completed',
  terminated: 'Terminated',
  planned: 'Planned',
  confirmed: 'Confirmed',
  cancelled: 'Cancelled',
  sent: 'Sent',
  accepted: 'Accepted',
  rejected: 'Rejected',
  expired: 'Expired',
  queued: 'Queued',
  failed: 'Failed',
}

export function StatusBadge({ status }: { status: string }) {
  return <Badge tone={tones[status] ?? 'neutral'}>{labels[status] ?? status}</Badge>
}
