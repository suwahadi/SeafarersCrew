import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { crewKeys, crewsApi, type Crew } from '@/api/crews'
import { Button } from '@/components/ui/Button'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors } from '@/lib/forms'
import { ContactFields, PersonalFields } from '../CrewFields'
import {
  CONTACT_FIELDS,
  contactFromCrew,
  contactSchema,
  PERSONAL_FIELDS,
  personalFromCrew,
  personalSchema,
  toCrewInput,
  type ContactValues,
  type PersonalValues,
} from '../crewSchema'

interface TabProps {
  crew: Crew
  onDirtyChange: (dirty: boolean) => void
}

function useSaveCrew(crew: Crew) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ReturnType<typeof toCrewInput>) => crewsApi.update(crew.id, input),
    onSuccess: (updated) => {
      qc.setQueryData(crewKeys.detail(crew.id), updated)
      void qc.invalidateQueries({ queryKey: ['grid', 'crews'] })
    },
  })
}

function FormError({ message }: { message: string | null }) {
  if (!message) return null
  return (
    <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
      {message}
    </p>
  )
}

export function PersonalTab({ crew, onDirtyChange }: TabProps) {
  const toast = useToast()
  const save = useSaveCrew(crew)
  const [formError, setFormError] = useState<string | null>(null)
  const form = useForm<PersonalValues>({ resolver: zodResolver(personalSchema), values: personalFromCrew(crew) })
  const dirty = form.formState.isDirty

  useEffect(() => onDirtyChange(dirty), [dirty, onDirtyChange])

  const onSubmit = form.handleSubmit(async (values) => {
    setFormError(null)
    try {
      await save.mutateAsync(toCrewInput(values, contactFromCrew(crew)))
      toast.success('Personal data saved.')
    } catch (err) {
      setFormError(applyServerErrors(err, form.setError, PERSONAL_FIELDS))
    }
  })

  return (
    <form onSubmit={onSubmit} noValidate className="space-y-4" aria-label="Personal data">
      <FormError message={formError} />
      <PersonalFields register={form.register} control={form.control} errors={form.formState.errors} age={crew.age} />
      <div className="flex justify-end gap-2 max-sm:*:flex-1">
        <Button variant="secondary" disabled={!dirty} onClick={() => form.reset(personalFromCrew(crew))}>
          Discard changes
        </Button>
        <Button type="submit" loading={save.isPending} disabled={!dirty}>
          Save personal data
        </Button>
      </div>
    </form>
  )
}

export function ContactTab({ crew, onDirtyChange }: TabProps) {
  const toast = useToast()
  const save = useSaveCrew(crew)
  const [formError, setFormError] = useState<string | null>(null)
  const form = useForm<ContactValues>({ resolver: zodResolver(contactSchema), values: contactFromCrew(crew) })
  const dirty = form.formState.isDirty

  useEffect(() => onDirtyChange(dirty), [dirty, onDirtyChange])

  const onSubmit = form.handleSubmit(async (values) => {
    setFormError(null)
    try {
      await save.mutateAsync(toCrewInput(personalFromCrew(crew), values))
      toast.success('Contact details saved.')
    } catch (err) {
      setFormError(applyServerErrors(err, form.setError, CONTACT_FIELDS))
    }
  })

  return (
    <form onSubmit={onSubmit} noValidate className="space-y-4" aria-label="Contact and address">
      <FormError message={formError} />
      <ContactFields register={form.register} errors={form.formState.errors} />
      <div className="flex justify-end gap-2 max-sm:*:flex-1">
        <Button variant="secondary" disabled={!dirty} onClick={() => form.reset(contactFromCrew(crew))}>
          Discard changes
        </Button>
        <Button type="submit" loading={save.isPending} disabled={!dirty}>
          Save contact details
        </Button>
      </div>
    </form>
  )
}
