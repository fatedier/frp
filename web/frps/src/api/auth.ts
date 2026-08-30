import { http } from './http'

export const loginRequest = (user: string, password: string) =>
  http.post('../api/auth/login', { user, password })

export const logoutRequest = () => http.post('../api/auth/logout')

export const checkAuthRequest = () =>
  http.get('../api/auth/check', { cache: 'no-store' })
