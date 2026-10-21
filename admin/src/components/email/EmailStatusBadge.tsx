import type { EmailStatus } from '@/api/email'
import { Badge, type Tone } from '../ui/Layout'

const tones: Record<EmailStatus, Tone> = { queued: 'yellow', sent: 'green', failed: 'red' }
const labels: Record<EmailStatus, string> = { queued: 'Queued', sent: 'Sent', failed: 'Failed' }

export function EmailStatusBadge({ status }: { status: EmailStatus }) {
  return <Badge tone={tones[status]}>{labels[status]}</Badge>
}
