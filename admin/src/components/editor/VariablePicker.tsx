import { useId } from 'react'
import type { DocumentKind, VariableGroup } from '@/api/templates'

export function tagText(tag: string): string {
  return `{${tag}}`
}

export function groupsForKind(groups: VariableGroup[], kind?: DocumentKind): VariableGroup[] {
  return kind ? groups.filter((g) => g.kinds.includes(kind)) : groups
}

export function tagsToGroups(tags: string[]): VariableGroup[] {
  return tags.length === 0 ? [] : [{ name: 'Available tags', kinds: [], variables: tags.map((tag) => ({ tag, description: '' })) }]
}

interface VariablePickerProps {
  groups: VariableGroup[]
  onInsert: (text: string) => void
  label?: string
}

export function VariablePicker({ groups, onInsert, label = 'Insert variable' }: VariablePickerProps) {
  const id = useId()
  return (
    <select
      id={id}
      aria-label={label}
      value=""
      onChange={(e) => {
        if (e.target.value !== '') onInsert(tagText(e.target.value))
      }}
      className="h-8 max-w-56 rounded-md border-0 bg-white text-sm text-slate-700 ring-1 ring-inset ring-slate-300 focus:ring-2 focus:ring-navy-500"
    >
      <option value="">{label}…</option>
      {groups.map((g) => (
        <optgroup key={g.name} label={g.name}>
          {g.variables.map((v) => (
            <option key={`${g.name}-${v.tag}`} value={v.tag}>
              {v.description ? `${tagText(v.tag)} — ${v.description}` : tagText(v.tag)}
            </option>
          ))}
        </optgroup>
      ))}
    </select>
  )
}
