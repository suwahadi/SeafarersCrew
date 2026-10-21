import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { DocumentTemplate, EmailTemplate } from '@/api/templates'
import { mockApi, renderApp } from '@/test/utils'
import { DocumentTemplateForm } from './DocumentTemplateEditPage'
import { EmailTemplateForm } from './EmailTemplateEditPage'

afterEach(() => {
  vi.unstubAllGlobals()
})

const email: EmailTemplate = {
  code: 'send_cv',
  name: 'Send CV',
  subject: 'CV for {crew_name}',
  body_html: '<p>Hello {crew_name}</p>',
  available_tags: ['crew_name', 'rank'],
  updated_at: '2026-01-01T00:00:00Z',
}

describe('email template editor', () => {
  it('renders the preview in a fully sandboxed iframe and lists unknown tags', async () => {
    const calls = mockApi(() => ({ body: { data: { subject: 'CV for Budi', body_html: '<p>Hello Budi {oops}</p>', unknown_tags: ['oops'] } } }))
    renderApp(<EmailTemplateForm template={email} />)
    await userEvent.click(screen.getByRole('button', { name: 'Preview' }))
    const frame = await screen.findByTitle('Email preview')
    expect(frame).toHaveAttribute('sandbox', '')
    expect(frame).toHaveAttribute('srcdoc', '<p>Hello Budi {oops}</p>')
    expect(screen.getByRole('alert')).toHaveTextContent('{oops}')
    expect(calls[0]).toMatchObject({ method: 'POST', path: '/email-templates/send_cv/preview' })
  })

  it('does not save an empty subject', async () => {
    const calls = mockApi(() => undefined)
    renderApp(<EmailTemplateForm template={email} />)
    await userEvent.clear(screen.getByLabelText(/^Subject/))
    await userEvent.click(screen.getByRole('button', { name: 'Save template' }))
    expect(await screen.findByText('The subject is required.')).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'PUT')).toBe(false)
  })
})

describe('document template editor', () => {
  const doc: DocumentTemplate = {
    id: 't1',
    name: 'Default SEA',
    kind: 'sea',
    paper_size: 'a4',
    orientation: 'portrait',
    header_html: '<p>Head</p>',
    body_html: '<p>Body</p>',
    footer_html: '<p>Foot</p>',
    is_default: true,
    updated_at: '2026-01-01T00:00:00Z',
  }
  const catalogue = { groups: [{ name: 'Crew', kinds: ['sea'], variables: [{ tag: 'crew_name', description: 'Full name' }] }], cv_sections: [] }

  it('saves with the locked kind and warns about unknown tags returned by the server', async () => {
    const calls = mockApi((c) => {
      if (c.path === '/document-templates/variables') return { body: { data: catalogue } }
      if (c.method === 'PUT') return { body: { data: { ...doc, unknown_tags: ['mystery'] } } }
      return undefined
    })
    renderApp(<DocumentTemplateForm template={doc} />)
    expect(screen.getByLabelText('Document kind')).toBeDisabled()
    await userEvent.click(screen.getByRole('button', { name: 'Save template' }))
    expect(await screen.findByText(/\{mystery\}/)).toBeInTheDocument()
    const put = calls.find((c) => c.method === 'PUT')
    expect(put?.path).toBe('/document-templates/t1')
    expect(put?.body).toMatchObject({ name: 'Default SEA', kind: 'sea', paper_size: 'a4', orientation: 'portrait', is_default: true, body_html: '<p>Body</p>' })
  })

  it('offers variables for the template kind in each editor', async () => {
    mockApi((c) => (c.path === '/document-templates/variables' ? { body: { data: catalogue } } : undefined))
    renderApp(<DocumentTemplateForm template={null} />)
    await waitFor(() => expect(screen.getAllByLabelText('Insert variable')[0]?.querySelectorAll('option').length).toBeGreaterThan(1))
    expect(screen.getAllByLabelText('Insert variable')).toHaveLength(3)
  })
})
