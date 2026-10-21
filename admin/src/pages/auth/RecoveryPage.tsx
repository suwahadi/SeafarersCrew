import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate } from 'react-router'
import { z } from 'zod'
import { authApi } from '@/api/auth'
import { SESSION_KEY } from '@/app/auth'
import { Button } from '@/components/ui/Button'
import { TextField } from '@/components/ui/Field'
import { applyServerErrors, PASSWORD_MIN } from '@/lib/forms'
import { AuthShell } from './AuthShell'

const schema = z
  .object({
    email: z.email('Enter a valid email address.'),
    code: z.string().trim().min(10, 'Enter a recovery code such as ABCDE-23456.'),
    new_password: z.string().min(PASSWORD_MIN, `The password must be at least ${PASSWORD_MIN} characters.`),
    confirm: z.string(),
  })
  .refine((v) => v.new_password === v.confirm, { path: ['confirm'], message: 'The passwords do not match.' })

type Values = z.infer<typeof schema>

export default function RecoveryPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [formError, setFormError] = useState<string | null>(null)
  const { register, handleSubmit, setError, formState } = useForm<Values>({ resolver: zodResolver(schema) })

  const onSubmit = handleSubmit(async (v) => {
    setFormError(null)
    try {
      const session = await authApi.recover(v.email, v.code, v.new_password)
      qc.setQueryData(SESSION_KEY, session)
      void navigate('/', { replace: true })
    } catch (err) {
      setFormError(applyServerErrors(err, setError, ['email', 'code', 'new_password']))
    }
  })

  return (
    <AuthShell title="Recover access" footer={<Link to="/login" className="text-navy-700 hover:underline">Back to sign in</Link>}>
      <p className="mb-4 text-sm text-slate-600">Each recovery code works once. After signing in, regenerate your codes from Settings.</p>
      <form onSubmit={onSubmit} noValidate className="space-y-4">
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <TextField label="Email" type="email" autoComplete="username" required error={formState.errors.email?.message} {...register('email')} />
        <TextField label="Recovery code" autoComplete="one-time-code" required error={formState.errors.code?.message} {...register('code')} />
        <TextField label="New password" type="password" autoComplete="new-password" required error={formState.errors.new_password?.message} {...register('new_password')} />
        <TextField label="Confirm new password" type="password" autoComplete="new-password" required error={formState.errors.confirm?.message} {...register('confirm')} />
        <Button type="submit" className="w-full" loading={formState.isSubmitting}>
          Recover and sign in
        </Button>
      </form>
    </AuthShell>
  )
}
