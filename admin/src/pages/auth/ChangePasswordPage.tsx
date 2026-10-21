import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router'
import { z } from 'zod'
import { authApi } from '@/api/auth'
import { SESSION_KEY } from '@/app/auth'
import { Button } from '@/components/ui/Button'
import { TextField } from '@/components/ui/Field'
import { applyServerErrors, PASSWORD_MIN } from '@/lib/forms'
import { AuthShell } from './AuthShell'

const schema = z
  .object({
    current_password: z.string().min(1, 'The current password is required.'),
    new_password: z.string().min(PASSWORD_MIN, `The password must be at least ${PASSWORD_MIN} characters.`),
    confirm: z.string(),
  })
  .refine((v) => v.new_password === v.confirm, { path: ['confirm'], message: 'The passwords do not match.' })

type Values = z.infer<typeof schema>

export default function ChangePasswordPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [formError, setFormError] = useState<string | null>(null)
  const { register, handleSubmit, setError, formState } = useForm<Values>({ resolver: zodResolver(schema) })

  const onSubmit = handleSubmit(async (v) => {
    setFormError(null)
    try {
      const session = await authApi.changePassword(v.current_password, v.new_password)
      qc.setQueryData(SESSION_KEY, session)
      void navigate('/', { replace: true })
    } catch (err) {
      setFormError(applyServerErrors(err, setError, ['current_password', 'new_password']))
    }
  })

  return (
    <AuthShell title="Choose a new password">
      <p className="mb-4 text-sm text-slate-600">You must set a new password before continuing. All other sessions will be signed out.</p>
      <form onSubmit={onSubmit} noValidate className="space-y-4">
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <TextField label="Current or temporary password" type="password" autoComplete="current-password" required error={formState.errors.current_password?.message} {...register('current_password')} />
        <TextField label="New password" type="password" autoComplete="new-password" required error={formState.errors.new_password?.message} {...register('new_password')} />
        <TextField label="Confirm new password" type="password" autoComplete="new-password" required error={formState.errors.confirm?.message} {...register('confirm')} />
        <Button type="submit" className="w-full" loading={formState.isSubmitting}>
          Save password
        </Button>
      </form>
    </AuthShell>
  )
}
