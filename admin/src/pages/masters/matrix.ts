import type { ComplianceCell, ComplianceMatrix } from '@/api/masters'

export type ColumnKind = 'document' | 'certificate'

export const cellKey = (rankId: string, kind: ColumnKind, typeId: string) => `${rankId}|${kind}|${typeId}`

export function cellsToSet(cells: ComplianceCell[]): Set<string> {
  const out = new Set<string>()
  for (const c of cells) {
    if (c.document_type_id) out.add(cellKey(c.rank_id, 'document', c.document_type_id))
    if (c.certificate_type_id) out.add(cellKey(c.rank_id, 'certificate', c.certificate_type_id))
  }
  return out
}

export function setToCells(set: Set<string>): ComplianceCell[] {
  return [...set].sort().map((key) => {
    const [rankId = '', kind, typeId = ''] = key.split('|')
    return {
      rank_id: rankId,
      document_type_id: kind === 'document' ? typeId : null,
      certificate_type_id: kind === 'certificate' ? typeId : null,
    }
  })
}

export function columnsOf(m: ComplianceMatrix): { kind: ColumnKind; id: string; name: string }[] {
  return [
    ...m.document_types.map((d) => ({ kind: 'document' as const, id: d.id, name: d.name })),
    ...m.certificate_types.map((c) => ({ kind: 'certificate' as const, id: c.id, name: c.stcw_reference ? `${c.name} (${c.stcw_reference})` : c.name })),
  ]
}

export function sameSet(a: Set<string>, b: Set<string>): boolean {
  if (a.size !== b.size) return false
  for (const k of a) if (!b.has(k)) return false
  return true
}
