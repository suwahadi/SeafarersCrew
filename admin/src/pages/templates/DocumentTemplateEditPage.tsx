import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { Link, useNavigate, useParams } from 'react-router'
import { z } from 'zod'
import { errorMessage } from '@/api/client'
import { DOCUMENT_TEMPLATES_KEY, documentTemplatesApi, type DocumentTemplate, type DocumentTemplateInput } from '@/api/templates'
import { RichTextEditor } from '@/components/editor/RichTextEditor'
import { groupsForKind, tagText } from '@/components/editor/VariablePicker'
import { Button } from '@/components/ui/Button'
import { CheckboxField, SelectField, TextField } from '@/components/ui/Field'
import { Card, PageHeader } from '@/components/ui/Layout'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors } from '@/lib/forms'
import { kindLabels } from './DocumentTemplatesPage'

const schema = z.object({
  name: z.string().trim().min(1, 'The name is required.').max(120, 'The name may not exceed 120 characters.'),
  kind: z.enum(['sea', 'quotation', 'cv']),
  paper_size: z.enum(['a4', 'letter']),
  orientation: z.enum(['portrait', 'landscape']),
  is_default: z.boolean(),
  header_html: z.string(),
  body_html: z.string(),
  footer_html: z.string(),
})

type Values = z.infer<typeof schema>

const kindOptions = (Object.keys(kindLabels) as (keyof typeof kindLabels)[]).map((k) => ({ value: k, label: kindLabels[k] }))
const fieldNames = ['name', 'kind', 'paper_size', 'orientation', 'is_default', 'header_html', 'body_html', 'footer_html'] as const

export default function DocumentTemplateEditPage() {
  const { id } = useParams()
  const template = useQuery({
    queryKey: [...DOCUMENT_TEMPLATES_KEY, id],
    queryFn: ({ signal }) => documentTemplatesApi.get(id ?? '', signal),
    enabled: id !== undefined,
  })

  return (
    <div>
      <PageHeader
        title={id ? 'Edit document template' : 'New document template'}
        description="Insert variables with the picker. They are replaced with real data when a document is generated."
        actions={
          <Link to="/document-templates" className="text-sm text-navy-700 hover:underline">
            Back to templates
          </Link>
        }
      />
      {id === undefined ? (
        <DocumentTemplateForm template={null} />
      ) : template.isPending ? (
        <div className="flex justify-center p-8">
          <Spinner />
        </div>
      ) : template.isError ? (
        <p role="alert" className="text-sm text-red-700">
          {errorMessage(template.error)}
        </p>
      ) : (
        <DocumentTemplateForm key={template.data.updated_at} template={template.data} />
      )}
    </div>
  )
}

function toValues(t: DocumentTemplate | null): Values {
  return {
    name: t?.name ?? '',
    kind: t?.kind ?? 'sea',
    paper_size: t?.paper_size ?? 'a4',
    orientation: t?.orientation ?? 'portrait',
    is_default: t?.is_default ?? false,
    header_html: t?.header_html ?? '',
    body_html: t?.body_html ?? '',
    footer_html: t?.footer_html ?? '',
  }
}

export function DocumentTemplateForm({ template }: { template: DocumentTemplate | null }) {
  const qc = useQueryClient()
  const toast = useToast()
  const navigate = useNavigate()
  const [formError, setFormError] = useState<string | null>(null)
  const [unknownTags, setUnknownTags] = useState<string[]>(template?.unknown_tags ?? [])
  const variables = useQuery({ queryKey: [...DOCUMENT_TEMPLATES_KEY, 'variables'], queryFn: ({ signal }) => documentTemplatesApi.variables(signal), staleTime: Infinity })
  const { register, control, handleSubmit, setError, watch, formState } = useForm<Values>({ resolver: zodResolver(schema), defaultValues: toValues(template) })
  const kind = watch('kind')
  const groups = useMemo(() => groupsForKind(variables.data?.groups ?? [], kind), [variables.data, kind])

  const save = useMutation({
    mutationFn: (v: Values) => {
      const body: DocumentTemplateInput = v
      return template ? documentTemplatesApi.update(template.id, body) : documentTemplatesApi.create(body)
    },
    onSuccess: async (saved) => {
      setFormError(null)
      setUnknownTags(saved.unknown_tags ?? [])
      await qc.invalidateQueries({ queryKey: DOCUMENT_TEMPLATES_KEY })
      toast.success('Template saved.')
      if (!template) void navigate(`/document-templates/${saved.id}`, { replace: true })
    },
    onError: (err) => setFormError(applyServerErrors(err, setError, fieldNames)),
  })

  const editor = (name: 'header_html' | 'body_html' | 'footer_html', label: string, minHeight: string) => (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => <RichTextEditor label={label} value={field.value} onChange={field.onChange} groups={groups} error={fieldState.error?.message} minHeight={minHeight} />}
    />
  )

  return (
    <form onSubmit={handleSubmit((v) => save.mutate(v))} noValidate className="space-y-6">
      {formError ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
          {formError}
        </p>
      ) : null}
      {unknownTags.length > 0 ? (
        <p role="alert" className="rounded-md bg-yellow-50 p-3 text-sm text-yellow-800">
          Unknown tags found: {unknownTags.map((t) => tagText(t)).join(', ')}. They will be left as typed in generated documents.
        </p>
      ) : null}
      <Card>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <TextField label="Name" required error={formState.errors.name?.message} {...register('name')} />
          <SelectField label="Document kind" options={kindOptions} disabled={template !== null} error={formState.errors.kind?.message} {...register('kind')} />
          <SelectField
            label="Paper size"
            options={[
              { value: 'a4', label: 'A4' },
              { value: 'letter', label: 'Letter' },
            ]}
            {...register('paper_size')}
          />
          <SelectField
            label="Orientation"
            options={[
              { value: 'portrait', label: 'Portrait' },
              { value: 'landscape', label: 'Landscape' },
            ]}
            {...register('orientation')}
          />
          <CheckboxField label="Default template for this kind" hint="The default is used when no template is chosen." {...register('is_default')} />
        </div>
      </Card>
      <div className="space-y-4">
        {editor('header_html', 'Header', '6rem')}
        {editor('body_html', 'Body', '20rem')}
        {editor('footer_html', 'Footer', '6rem')}
      </div>
      <div className="flex gap-2">
        <Button type="submit" loading={save.isPending}>
          Save template
        </Button>
        <Link to="/document-templates" className="inline-flex h-9 items-center rounded-md px-4 text-sm text-slate-700 hover:bg-slate-100">
          Cancel
        </Link>
      </div>
    </form>
  )
}
