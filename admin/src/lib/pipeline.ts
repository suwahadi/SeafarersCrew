import type { Applicant, Stage, StageCounts } from '@/api/applicants'

export const STAGES: Stage[] = ['new', 'screening', 'interview', 'accepted', 'rejected']

export const STAGE_LABELS: Record<Stage, string> = {
  new: 'New',
  screening: 'Screening',
  interview: 'Interview',
  accepted: 'Accepted',
  rejected: 'Rejected',
}

export const NOTE_MAX = 1000

export function isStage(value: string): value is Stage {
  return (STAGES as string[]).includes(value)
}

export function noteRequired(stage: Stage): boolean {
  return stage === 'rejected'
}

export function validateStageNote(stage: Stage, note: string): string | null {
  const trimmed = note.trim()
  if (noteRequired(stage) && trimmed === '') return 'A note is required when rejecting an applicant.'
  if (trimmed.length > NOTE_MAX) return `The note must be at most ${NOTE_MAX} characters.`
  return null
}

export function emptyCounts(): StageCounts {
  return { new: 0, screening: 0, interview: 0, accepted: 0, rejected: 0 }
}

export function groupByStage(applicants: Applicant[]): Record<Stage, Applicant[]> {
  const groups: Record<Stage, Applicant[]> = { new: [], screening: [], interview: [], accepted: [], rejected: [] }
  for (const a of applicants) groups[a.stage].push(a)
  return groups
}

export function moveTargets(current: Stage): Stage[] {
  return STAGES.filter((s) => s !== current)
}
