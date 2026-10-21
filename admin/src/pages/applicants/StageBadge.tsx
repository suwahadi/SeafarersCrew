import type { Stage } from '@/api/applicants'
import { Badge, type Tone } from '@/components/ui/Layout'
import { STAGE_LABELS } from '@/lib/pipeline'

const tones: Record<Stage, Tone> = {
  new: 'blue',
  screening: 'yellow',
  interview: 'orange',
  accepted: 'green',
  rejected: 'red',
}

export function StageBadge({ stage }: { stage: Stage }) {
  return <Badge tone={tones[stage]}>{STAGE_LABELS[stage]}</Badge>
}
