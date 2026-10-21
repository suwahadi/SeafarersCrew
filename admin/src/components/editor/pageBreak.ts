import { Node } from '@tiptap/core'

export const PageBreak = Node.create({
  name: 'pageBreak',
  group: 'block',
  atom: true,
  selectable: true,
  parseHTML() {
    return [{ tag: 'div.page-break' }]
  },
  renderHTML() {
    return ['div', { class: 'page-break' }]
  },
})
