import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useRef, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { Link, useParams } from 'react-router'
import { z } from 'zod'
import { errorMessage } from '@/api/client'
import { EMAIL_TEMPLATES_KEY, emailTemplatesApi, type EmailPreview, type EmailTemplate } from '@/api/templates'
import { RichTextEditor } from '@/components/editor/RichTextEditor'
import { tagsToGroups, tagText, VariablePicker } from '@/components/editor/VariablePicker'
import { Button } from '@/components/ui/Button'
import { TextField } from '@/components/ui/Field'
import { Card, PageHeader } from '@/components/ui/Layout'
import { Spinner } from '@/components/ui/Spinner'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors } from '@/lib/forms'

const schema = z.object({
  subject: z.string().trim().min(1, 'The subject is required.').max(200, 'The subject may not exceed 200 characters.'),
  body_html: z.string().refine((v) => v.replace(/<[^>]*>/g, '').trim() !== '', 'The body is required.'),
})

type Values = z.infer<typeof schema>

export default function EmailTemplateEditPage() {
  const { code = '' } = useParams()
  const template = useQuery({ queryKey: [...EMAIL_TEMPLATES_KEY, code], queryFn: ({ signal }) => emailTemplatesApi.get(code, signal) })

  return (
    <div>
      <PageHeader
        title={template.data ? template.data.name : 'Email template'}
        description="Use {tags} to insert values. Unknown tags are flagged in the preview."
        actions={
          <Link to="/email-templates" className="text-sm text-navy-700 hover:underline">
            Back to templates
          </Link>
        }
      />
      {template.isPending ? (
        <div className="flex justify-center p-8">
          <Spinner />
        </div>
      ) : template.isError ? (
        <p role="alert" className="text-sm text-red-700">
          {errorMessage(template.error)}
        </p>
      ) : (
        <EmailTemplateForm key={template.data.updated_at} template={template.data} />
      )}
    </div>
  )
}

export function EmailTemplateForm({ template }: { template: EmailTemplate }) {
  const qc = useQueryClient()
  const toast = useToast()
  const subjectRef = useRef<HTMLInputElement | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [preview, setPreview] = useState<EmailPreview | null>(null)
  const groups = useMemo(() => tagsToGroups(template.available_tags ?? []), [template.available_tags])
  const { register, control, handleSubmit, getValues, setValue, setError, trigger, formState } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { subject: template.subject, body_html: template.body_html ?? '' },
  })
  const subjectField = register('subject')

  const save = useMutation({
    mutationFn: (v: Values) => emailTemplatesApi.update(template.code, v.subject, v.body_html),
    onSuccess: async (saved) => {
      setFormError(null)
      qc.setQueryData([...EMAIL_TEMPLATES_KEY, template.code], { ...saved, available_tags: saved.available_tags ?? template.available_tags })
      await qc.invalidateQueries({ queryKey: EMAIL_TEMPLATES_KEY, exact: true })
      toast.success('Email template saved.')
    },
    onError: (err) => setFormError(applyServerErrors(err, setError, ['subject', 'body_html'])),
  })

  const render = useMutation({
    mutationFn: (v: Values) => emailTemplatesApi.preview(template.code, v.subject, v.body_html),
    onSuccess: setPreview,
    onError: (err) => toast.error(errorMessage(err)),
  })

  const insertIntoSubject = (text: string) => {
    const input = subjectRef.current
    const current = getValues('subject')
    const start = input?.selectionStart ?? current.length
    const end = input?.selectionEnd ?? current.length
    setValue('subject', current.slice(0, start) + text + current.slice(end), { shouldDirty: true })
    input?.focus()
  }

  const onPreview = async () => {
    if (await trigger()) render.mutate(getValues())
  }

  return (
    <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
      <form onSubmit={handleSubmit((v) => save.mutate(v))} noValidate className="space-y-4">
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <div className="space-y-1">
          <TextField
            label="Subject"
            required
            error={formState.errors.subject?.message}
            {...subjectField}
            ref={(el) => {
              subjectField.ref(el)
              subjectRef.current = el
            }}
          />
          {groups.length > 0 ? <VariablePicker groups={groups} label="Insert tag in subject" onInsert={insertIntoSubject} /> : null}
        </div>
        <Controller
          control={control}
          name="body_html"
          render={({ field, fieldState }) => (
            <RichTextEditor label="Body" value={field.value} onChange={field.onChange} groups={groups} error={fieldState.error?.message} minHeight="16rem" />
          )}
        />
        {groups.length > 0 ? (
          <p className="text-xs text-slate-500">
            Available tags: {(template.available_tags ?? []).map((t) => tagText(t)).join(' ')}
          </p>
        ) : null}
        <div className="flex gap-2">
          <Button type="submit" loading={save.isPending}>
            Save template
          </Button>
          <Button variant="secondary" loading={render.isPending} onClick={() => void onPreview()}>
            Preview
          </Button>
        </div>
      </form>
      <Card title="Preview">
        {preview ? (
          <div className="space-y-3">
            {preview.unknown_tags.length > 0 ? (
              <p role="alert" className="rounded-md bg-yellow-50 p-3 text-sm text-yellow-800">
                Unknown tags: {preview.unknown_tags.map((t) => tagText(t)).join(', ')}. They will not be replaced when the email is sent.
              </p>
            ) : null}
            <p className="text-sm">
              <span className="font-medium text-slate-700">Subject: </span>
              {preview.subject}
            </p>
            <iframe title="Email preview" sandbox="" srcDoc={preview.body_html} className="h-96 w-full rounded-md border border-slate-200 bg-white" />
          </div>
        ) : (
          <p className="text-sm text-slate-500">Select Preview to render this template with sample values.</p>
        )}
      </Card>
    </div>
  )
}
