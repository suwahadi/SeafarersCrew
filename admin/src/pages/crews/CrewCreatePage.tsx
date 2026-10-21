import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type SyntheticEvent } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router'
import { crewKeys, crewsApi } from '@/api/crews'
import { Button } from '@/components/ui/Button'
import { Card, PageHeader } from '@/components/ui/Layout'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors, unmatchedError } from '@/lib/forms'
import { ContactFields, PersonalFields } from './CrewFields'
import {
  CONTACT_FIELDS,
  contactSchema,
  EMPTY_CONTACT,
  EMPTY_PERSONAL,
  PERSONAL_FIELDS,
  personalSchema,
  toCrewInput,
  type ContactValues,
  type PersonalValues,
} from './crewSchema'

export default function CrewCreatePage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const toast = useToast()
  const [formError, setFormError] = useState<string | null>(null)
  const personal = useForm<PersonalValues>({ resolver: zodResolver(personalSchema), defaultValues: EMPTY_PERSONAL })
  const contact = useForm<ContactValues>({ resolver: zodResolver(contactSchema), defaultValues: EMPTY_CONTACT })
  const create = useMutation({ mutationFn: crewsApi.create })

  const onSubmit = async (e: SyntheticEvent) => {
    e.preventDefault()
    setFormError(null)
    const [okPersonal, okContact] = await Promise.all([personal.trigger(), contact.trigger()])
    if (!okPersonal || !okContact) return
    try {
      const crew = await create.mutateAsync(toCrewInput(personal.getValues(), contact.getValues()))
      void qc.invalidateQueries({ queryKey: ['grid', 'crews'] })
      qc.setQueryData(crewKeys.detail(crew.id), crew)
      toast.success(`Crew ${crew.id} created.`)
      void navigate(`/crews/${crew.id}`, { replace: true })
    } catch (err) {
      applyServerErrors(err, personal.setError, PERSONAL_FIELDS)
      applyServerErrors(err, contact.setError, CONTACT_FIELDS)
      setFormError(unmatchedError(err, [...PERSONAL_FIELDS, ...CONTACT_FIELDS]))
    }
  }

  return (
    <>
      <PageHeader title="New crew" description="A unique 9-character Crew ID is assigned on save. The crew member starts as Available." />
      <form onSubmit={(e) => void onSubmit(e)} noValidate className="space-y-4">
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <Card title="Personal data">
          <PersonalFields register={personal.register} control={personal.control} errors={personal.formState.errors} />
        </Card>
        <Card title="Contact & address">
          <ContactFields register={contact.register} errors={contact.formState.errors} />
        </Card>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => void navigate('/crews')}>
            Cancel
          </Button>
          <Button type="submit" loading={create.isPending}>
            Create crew
          </Button>
        </div>
      </form>
    </>
  )
}
