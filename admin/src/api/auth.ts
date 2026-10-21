import { api, setCsrfToken } from './client'

export interface Admin {
  id: string
  name: string
  email: string
  must_change_password: boolean
  password_changed_at: string
}

export interface Session {
  admin: Admin
  csrf_token: string
}

function remember(session: Session): Session {
  setCsrfToken(session.csrf_token)
  return session
}

export const authApi = {
  me: () => api.get<Session>('/me').then(remember),
  login: (email: string, password: string) => api.post<Session>('/auth/login', { email, password }).then(remember),
  recover: (email: string, code: string, newPassword: string) =>
    api.post<Session>('/auth/recovery', { email, code, new_password: newPassword }).then(remember),
  logout: () => api.post('/auth/logout'),
  changePassword: (currentPassword: string, newPassword: string) =>
    api.put<Session>('/me/password', { current_password: currentPassword, new_password: newPassword }).then(remember),
  updateProfile: (name: string, email: string) => api.put<Admin>('/me', { name, email }),
  regenerateRecoveryCodes: () => api.post<{ codes: string[] }>('/recovery-codes/regenerate'),
}
