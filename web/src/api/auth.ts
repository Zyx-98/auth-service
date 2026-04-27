import axios from 'axios'
import type { AxiosInstance } from 'axios'

const getApiUrl = (): string => {
  if (import.meta.env.VITE_API_URL) {
    return import.meta.env.VITE_API_URL
  }

  const { protocol, hostname, port } = window.location
  return `${protocol}//${hostname}${port ? `:${port}` : ''}`
}

const apiUrl = getApiUrl()

const client: AxiosInstance = axios.create({
  baseURL: apiUrl,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add access token
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor to handle token refresh
client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true

      try {
        const refreshToken = localStorage.getItem('refresh_token')
        if (!refreshToken) {
          throw new Error('No refresh token')
        }

        const response = await axios.post(`${apiUrl}/auth/refresh`, {
          refresh_token: refreshToken,
        })

        const { access_token, refresh_token } = response.data.data
        localStorage.setItem('access_token', access_token)
        localStorage.setItem('refresh_token', refresh_token)

        originalRequest.headers.Authorization = `Bearer ${access_token}`
        return client(originalRequest)
      } catch {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        window.location.href = '/login'
      }
    }

    return Promise.reject(error)
  }
)

export const authApi = {
  register: (email: string, password: string, passwordConfirm: string) =>
    client.post('/auth/register', { email, password, password_confirm: passwordConfirm }),

  login: (email: string, password: string) =>
    client.post('/auth/login', { email, password }),

  refreshToken: (refreshToken: string) =>
    client.post('/auth/refresh', { refresh_token: refreshToken }),

  logout: () =>
    client.post('/auth/logout'),

  getProfile: () =>
    client.get('/auth/me'),

  introspect: (token: string) =>
    client.post('/auth/introspect', { token }),

  setupTwoFA: () =>
    client.post('/auth/2fa/setup'),

  verifyTwoFA: (code: string) =>
    client.post('/auth/2fa/verify', { code }),

  verifyTwoFALogin: (code: string) =>
    client.post('/auth/2fa/verify-login', { code }),

  disableTwoFA: (code: string) =>
    client.post('/auth/2fa/disable', { code }),

  googleLoginRedirect: () =>
    client.get('/auth/login/google'),

  googleCallback: (code: string, state: string) =>
    client.post('/auth/callback/google', { code, state }),

  verifyOAuthTOTP: (code: string, totpToken: string) =>
    client.post('/auth/verify-oauth-totp', { code, totp_token: totpToken }),
}

export default client
