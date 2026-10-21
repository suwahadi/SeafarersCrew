import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, Navigate, useLocation, useNavigate } from 'react-router'
import { z } from 'zod'
import { authApi } from '@/api/auth'
import { SESSION_KEY, useAuth } from '@/app/auth'
import { Button } from '@/components/ui/Button'
import { TextField } from '@/components/ui/Field'
import { applyServerErrors } from '@/lib/forms'
import { AuthShell } from './AuthShell'

const schema = z.object({
  email: z.email('Enter a valid email address.'),
  password: z.string().min(1, 'The password is required.'),
})

type Values = z.infer<typeof schema>

export default function LoginPage() {
  const { admin } = useAuth()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const location = useLocation()
  const [formError, setFormError] = useState<string | null>(null)
  const { register, handleSubmit, setError, formState } = useForm<Values>({ resolver: zodResolver(schema), defaultValues: { email: '', password: '' } })

  if (admin) return <Navigate to="/" replace />

  const from = (location.state as { from?: string } | null)?.from ?? '/'
  const onSubmit = handleSubmit(async (values) => {
    setFormError(null)
    try {
      const session = await authApi.login(values.email, values.password)
      qc.setQueryData(SESSION_KEY, session)
      void navigate(session.admin.must_change_password ? '/change-password' : from, { replace: true })
    } catch (err) {
      setFormError(applyServerErrors(err, setError, ['email', 'password']))
    }
  })

  return (
    <AuthShell title="Sign in" footer={<Link to="/recovery" className="text-navy-700 hover:underline">Use a recovery code</Link>}>
      <form onSubmit={onSubmit} noValidate className="space-y-4">
        {formError ? (
          <p role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {formError}
          </p>
        ) : null}
        <TextField label="Email" type="email" autoComplete="username" required error={formState.errors.email?.message} {...register('email')} />
        <TextField label="Password" type="password" autoComplete="current-password" required error={formState.errors.password?.message} {...register('password')} />
        <Button type="submit" className="w-full" loading={formState.isSubmitting}>
          Sign in
        </Button>
      </form>
    </AuthShell>
  )
}
