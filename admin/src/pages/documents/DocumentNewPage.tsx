import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router'
import { errorMessage } from '@/api/client'
import { documentsApi, KIND_LABELS, RELATED_BY_KIND, type DocumentKind, type GeneratedDocument } from '@/api/documents'
import { openFile } from '@/api/files'
import { PreviewFrame } from '@/components/documents/PreviewFrame'
import { EmailSendDialog } from '@/components/email/EmailSendDialog'
import { Button } from '@/components/ui/Button'
import { CheckboxField, SelectField, TextField } from '@/components/ui/Field'
import { Card, PageHeader } from '@/components/ui/Layout'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { useDebounced } from '@/hooks/useDebounced'
import { titleCase } from '@/lib/format'

const EMAIL_TEMPLATE: Record<DocumentKind, string> = { sea: 'send_contract', quotation: 'send_quotation', cv: 'send_cv' }
const PUBLIC_ID = /^[A-Za-z0-9]{9}$/

function parseKind(value: string | null): DocumentKind {
  return value === 'quotation' || value === 'cv' ? value : 'sea'
}

const RELATED_LABEL: Record<DocumentKind, string> = { sea: 'Contract ID', quotation: 'Quotation ID', cv: 'Crew ID' }

export default function DocumentNewPage() {
  const [params] = useSearchParams()
  const qc = useQueryClient()
  const toast = useToast()
  const [kind, setKind] = useState<DocumentKind>(parseKind(params.get('kind')))
  const [relatedId, setRelatedId] = useState((params.get('related_id') ?? '').toUpperCase())
  const [templateId, setTemplateId] = useState('')
  const [hide, setHide] = useState<string[]>([])
  const [generated, setGenerated] = useState<GeneratedDocument | null>(null)
  const [emailOpen, setEmailOpen] = useState(false)

  const templates = useQuery({ queryKey: ['document-templates', kind], queryFn: () => documentsApi.templates(kind) })
  const variables = useQuery({ queryKey: ['document-variables'], queryFn: documentsApi.variables, staleTime: 5 * 60_000, enabled: kind === 'cv' })

  const debouncedId = useDebounced(relatedId.trim(), 400)
  const ready = PUBLIC_ID.test(debouncedId)
  const input = { kind, related_id: debouncedId, ...(templateId ? { template_id: templateId } : {}), ...(kind === 'cv' ? { hide } : {}) }
  const preview = useQuery({
    queryKey: ['document-preview', input],
    queryFn: () => documentsApi.preview(input),
    enabled: ready,
    retry: false,
  })

  const generate = useMutation({
    mutationFn: () => documentsApi.generate({ ...input, related_id: relatedId.trim() }),
    onSuccess: (doc) => {
      setGenerated(doc)
      void qc.invalidateQueries({ queryKey: ['grid', 'documents'] })
      void qc.invalidateQueries({ queryKey: ['generated-documents', RELATED_BY_KIND[kind]] })
      toast.success('PDF generated.')
      void openFile(doc.file_id).catch(() => undefined)
    },
    onError: (err) => toast.error(errorMessage(err)),
  })

  const cvSections = variables.data?.cv_sections ?? []
  const unknown = preview.data?.unknown_tags ?? []

  return (
    <>
      <PageHeader
        title="Generate document"
        description="Choose a template, check the preview and generate the PDF."
        actions={
          <Link to="/documents" className="text-sm text-navy-700 hover:underline">
            Back to documents
          </Link>
        }
      />
      <div className="grid gap-6 lg:grid-cols-[20rem_1fr]">
        <Card title="Options">
          <div className="space-y-4">
            <SelectField
              label="Document type"
              value={kind}
              onChange={(e) => {
                setKind(parseKind(e.target.value))
                setTemplateId('')
                setHide([])
                setGenerated(null)
              }}
              options={Object.entries(KIND_LABELS).map(([value, label]) => ({ value, label }))}
            />
            <TextField
              label={RELATED_LABEL[kind]}
              value={relatedId}
              maxLength={9}
              onChange={(e) => {
                setRelatedId(e.target.value.toUpperCase())
                setGenerated(null)
              }}
              hint="The 9-character ID shown on the record."
            />
            <SelectField
              label="Template"
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
              placeholder="Default template"
              options={(templates.data ?? []).map((t) => ({ value: t.id, label: t.is_default ? `${t.name} (default)` : t.name }))}
            />
            {kind === 'cv' && cvSections.length > 0 ? (
              <fieldset className="space-y-2">
                <legend className="text-sm font-medium text-slate-700">Hide sections</legend>
                {cvSections.map((section) => (
                  <CheckboxField
                    key={section}
                    label={titleCase(section)}
                    checked={hide.includes(section)}
                    onChange={(e) => setHide((list) => (e.target.checked ? [...list, section] : list.filter((s) => s !== section)))}
                  />
                ))}
              </fieldset>
            ) : null}
            <Button onClick={() => generate.mutate()} loading={generate.isPending} disabled={!PUBLIC_ID.test(relatedId.trim())} className="w-full">
              {generate.isPending ? 'Generating PDF…' : 'Generate PDF'}
            </Button>
            {generated ? (
              <div className="space-y-2 rounded-md bg-green-50 p-3 text-sm text-green-800" role="status">
                <p>PDF {generated.id} is ready.</p>
                <div className="flex gap-2">
                  <Button size="sm" variant="secondary" onClick={() => void openFile(generated.file_id).catch((e: unknown) => toast.error(errorMessage(e)))}>
                    Open PDF
                  </Button>
                  <Button size="sm" onClick={() => setEmailOpen(true)}>
                    Send by email
                  </Button>
                </div>
              </div>
            ) : null}
          </div>
        </Card>
        <Card title="Preview" actions={preview.isFetching ? <Spinner size="sm" label="Rendering preview" /> : undefined}>
          {unknown.length > 0 ? (
            <p role="alert" className="mb-3 rounded-md bg-yellow-50 p-3 text-sm text-yellow-800">
              The template uses unknown tags that will stay empty: {unknown.map((t) => `{${t}}`).join(', ')}.
            </p>
          ) : null}
          {!ready ? <p className="text-sm text-slate-500">Enter a valid {RELATED_LABEL[kind]} to see the preview.</p> : null}
          {ready && preview.isError ? (
            <p role="alert" className="text-sm text-red-700">
              {errorMessage(preview.error)}
            </p>
          ) : null}
          {ready && preview.data ? <PreviewFrame html={preview.data.html} /> : null}
        </Card>
      </div>
      {generated ? (
        <EmailSendDialog
          open={emailOpen}
          onClose={() => setEmailOpen(false)}
          relatedType={RELATED_BY_KIND[kind]}
          relatedId={relatedId.trim()}
          defaultTemplate={EMAIL_TEMPLATE[kind]}
          attachments={[{ fileId: generated.file_id, name: `${KIND_LABELS[kind]} (${generated.id})` }]}
        />
      ) : null}
    </>
  )
}
