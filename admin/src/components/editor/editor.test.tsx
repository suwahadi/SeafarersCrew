import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import type { VariableGroup } from '@/api/templates'
import { isSafeUrl, normalizeHtml, RichTextEditor } from './RichTextEditor'
import { groupsForKind, tagsToGroups, VariablePicker } from './VariablePicker'

const groups: VariableGroup[] = [
  { name: 'Crew', kinds: ['sea', 'cv'], variables: [{ tag: 'crew_name', description: 'Full name' }] },
  { name: 'Quotation', kinds: ['quotation'], variables: [{ tag: 'grand_total', description: 'Grand total' }] },
]

describe('VariablePicker', () => {
  it('inserts the chosen tag wrapped in braces', async () => {
    const onInsert = vi.fn()
    render(<VariablePicker groups={groups} onInsert={onInsert} />)
    await userEvent.selectOptions(screen.getByLabelText('Insert variable'), 'crew_name')
    expect(onInsert).toHaveBeenCalledWith('{crew_name}')
  })

  it('filters groups by document kind and maps plain tags to a group', () => {
    expect(groupsForKind(groups, 'quotation').map((g) => g.name)).toEqual(['Quotation'])
    expect(groupsForKind(groups).length).toBe(2)
    expect(tagsToGroups(['a', 'b'])[0]?.variables.map((v) => v.tag)).toEqual(['a', 'b'])
    expect(tagsToGroups([])).toEqual([])
  })
})

describe('RichTextEditor', () => {
  function Harness({ onValue }: { onValue: (v: string) => void }) {
    const [value, setValue] = useState('<p>Dear </p>')
    return (
      <RichTextEditor
        label="Body"
        value={value}
        groups={groups}
        onChange={(v) => {
          setValue(v)
          onValue(v)
        }}
      />
    )
  }

  it('inserts a variable tag at the cursor through the picker', async () => {
    const onValue = vi.fn()
    render(<Harness onValue={onValue} />)
    await userEvent.selectOptions(screen.getByLabelText('Insert variable'), 'crew_name')
    await waitFor(() => expect(onValue).toHaveBeenCalled())
    expect(onValue.mock.calls.at(-1)?.[0]).toContain('{crew_name}')
  })

  it('inserts the page break node as a page-break div', async () => {
    const onValue = vi.fn()
    render(<Harness onValue={onValue} />)
    await userEvent.click(screen.getByRole('button', { name: 'Insert page break' }))
    await waitFor(() => expect(onValue).toHaveBeenCalled())
    expect(onValue.mock.calls.at(-1)?.[0]).toContain('<div class="page-break"></div>')
  })

  it('treats an empty paragraph as empty content and rejects unsafe link targets', () => {
    expect(normalizeHtml('<p></p>')).toBe('')
    expect(isSafeUrl('https://example.com')).toBe(true)
    expect(isSafeUrl('javascript:alert(1)')).toBe(false)
  })
})
