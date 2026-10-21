import type { CrewStatusBadge } from '@/api/crews'
import { Badge } from '@/components/ui/Layout'

export function StatusBadge({ status }: { status: CrewStatusBadge }) {
  return <Badge color={status.color}>{status.name}</Badge>
}

export function ComplianceBadge({ compliant }: { compliant: boolean }) {
  return compliant ? <Badge tone="green">Compliant</Badge> : <Badge tone="red">Not compliant</Badge>
}
