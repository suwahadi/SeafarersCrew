import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { authApi, type Admin, type Session } from '@/api/auth'
import { SESSION_KEY, useAuth } from '@/app/auth'
import { Button } from '@/components/ui/Button'
import { TextField } from '@/components/ui/Field'
import { Card, PageHeader } from '@/components/ui/Layout'
import { useToast } from '@/components/ui/Toast'
import { applyServerErrors, PASSWORD_MIN } from '@/lib/forms'

const profileSchema = z.object({
  name: z.string().trim().min(1, 'The name is required.').max(120, 'The name may not exceed 120 characters.'),
  email: z.email('Enter a valid email address.').max(254, 'The email may not exceed 254 characters.'),
})

export const passwordSchema = z
  .object({
    current_password: z.string().min(1, 'The current password is required.'),
    new_password: z.string().min(PASSWORD_MIN, `The password must be at least ${PASSWORD_MIN} characters.`),
    confirm: z.string(),
  })
  .refine((v) => v.new_password === v.confirm, { path: ['confirm'], message: 'The passwords do not match.' })

type ProfileValues = z.infer<typeof profileSchema>
type PasswordValues = z.infer<typeof passwordSchema>

export default function AccountPage() {
  const { admin } = useAuth()
  return (
    <div>
      <PageHeader title="Account" description="Manage your profile and password." />
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {admin ? <ProfileCard admin={admin} /> : null}
        <PasswordCard />
      </div>
    </div>
  )
}

export function ProfileCard({ admin }: { admin: Admin }) {
  const qc = useQueryClient()
  const toast = useToast()
  const [formError, setFormError] = useState<string | null>(null)
  const { register, handleSubmit, setError, formState } = useForm<ProfileValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: { name: admin.name, email: admin.email },
  })

  const save = useMutation({
    mutationFn: (v: ProfileValues) => authApi.updateProfile(v.name, v.email),
    onSuccess: (updated) => {
      setFormError(null)
      qc.setQueryData<Session | null>(SESSION_KEY, (cur) => (cur ? { ...cur, admin: updated } : cur))
      toast.success('Profile saved.')
    },
    onError: (err) => setFormError(applyServerErrors(err, setError, ['name', 'email'])),
  })

  return (
    <Card title="Profile">
      <form onSubmit={handleSubmit((v) => save.mutate(v))} noValidate className="space-y-4">
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <TextField label="Name" autoComplete="name" required error={formState.errors.name?.message} {...register('name')} />
        <TextField label="Email" type="email" autoComplete="email" required error={formState.errors.email?.message} {...register('email')} />
        <Button type="submit" loading={save.isPending} disabled={!formState.isDirty}>
          Save profile
        </Button>
      </form>
    </Card>
  )
}

export function PasswordCard() {
  const qc = useQueryClient()
  const toast = useToast()
  const [formError, setFormError] = useState<string | null>(null)
  const { register, handleSubmit, setError, reset, formState } = useForm<PasswordValues>({ resolver: zodResolver(passwordSchema) })

  const change = useMutation({
    mutationFn: (v: PasswordValues) => authApi.changePassword(v.current_password, v.new_password),
    onSuccess: (session) => {
      setFormError(null)
      qc.setQueryData(SESSION_KEY, session)
      reset({ current_password: '', new_password: '', confirm: '' })
      toast.success('Password changed. Other sessions were signed out.')
    },
    onError: (err) => setFormError(applyServerErrors(err, setError, ['current_password', 'new_password'])),
  })

  return (
    <Card title="Change password">
      <form onSubmit={handleSubmit((v) => change.mutate(v))} noValidate className="space-y-4">
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <TextField label="Current password" type="password" autoComplete="current-password" required error={formState.errors.current_password?.message} {...register('current_password')} />
        <TextField label="New password" type="password" autoComplete="new-password" required error={formState.errors.new_password?.message} {...register('new_password')} />
        <TextField label="Confirm new password" type="password" autoComplete="new-password" required error={formState.errors.confirm?.message} {...register('confirm')} />
        <Button type="submit" loading={change.isPending}>
          Change password
        </Button>
      </form>
    </Card>
  )
}
