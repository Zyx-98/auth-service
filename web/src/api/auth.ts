import axios from 'axios'
import type { AxiosInstance, InternalAxiosRequestConfig } from 'axios'
import { getAccessToken, clearAuthCookies } from '../utils/cookie'

interface AuthRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
  skipAuthRefresh?: boolean
  skipAuthRedirect?: boolean
}

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
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add access token from cookie
client.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  config.withCredentials = true
  return config
})

// Response interceptor to handle token refresh
client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config as AuthRequestConfig | undefined

    if (
      error.response?.status === 401 &&
      originalRequest &&
      !originalRequest._retry &&
      !originalRequest.skipAuthRefresh &&
      originalRequest.url !== '/api/v1/auth/logout' &&
      originalRequest.url !== '/api/v1/auth/refresh'
    ) {
      originalRequest._retry = true

      try {
        const response = await axios.post(`${apiUrl}/api/v1/auth/refresh`, {
          refresh_token: '',
        }, {
          withCredentials: true
        })

        originalRequest.headers.Authorization = `Bearer ${response.data.data.access_token}`
        return client(originalRequest)
      } catch (err) {
        clearAuthCookies()
        if (!originalRequest.skipAuthRedirect && window.location.pathname !== '/login') {
          window.location.href = '/login'
        }
      }
    }

    return Promise.reject(error)
  }
)

export const authApi = {
  register: (email: string, password: string, passwordConfirm: string) =>
    client.post('/api/v1/auth/register', { email, password, password_confirm: passwordConfirm }),

  login: (email: string, password: string, deviceToken?: string) =>
    client.post('/api/v1/auth/login', {
      email,
      password,
      ...(deviceToken && { device_token: deviceToken })
    }),

  refreshToken: (refreshToken: string) =>
    client.post('/api/v1/auth/refresh', { refresh_token: refreshToken }),

  logout: () =>
    client.post('/api/v1/auth/logout', undefined, { skipAuthRefresh: true } as AuthRequestConfig),

  getProfile: (config?: Partial<AuthRequestConfig>) =>
    client.get('/api/v1/auth/me', config),

  introspect: (token: string) =>
    client.post('/api/v1/auth/introspect', { token }),

  setupTwoFA: () =>
    client.post('/api/v1/auth/2fa/setup'),

  verifyTwoFA: (code: string) =>
    client.post('/api/v1/auth/2fa/verify', { code }),

  verifyTwoFALogin: (code: string, trustDevice: boolean = false) => {
    const tempToken = sessionStorage.getItem('temp_token')
    return client.post('/api/v1/auth/2fa/verify-login', { code, trust_device: trustDevice }, {
      headers: {
        Authorization: `Bearer ${tempToken}`
      }
    })
  },

  disableTwoFA: (code: string) =>
    client.post('/api/v1/auth/2fa/disable', { code }),

  googleLoginRedirect: (deviceToken?: string) =>
    client.post('/api/v1/auth/login/google', {
      ...(deviceToken && { device_token: deviceToken })
    }),

  googleCallback: (code: string, state: string) =>
    client.post('/api/v1/auth/callback/google', { code, state }, { skipAuthRefresh: true } as AuthRequestConfig),

  verifyOAuthTOTP: (code: string, totpToken: string, trustDevice: boolean = false) =>
    client.post('/api/v1/auth/verify-oauth-totp', { code, totp_token: totpToken, trust_device: trustDevice }),

  getTrustedDevices: () =>
    client.get('/api/v1/auth/trusted-devices'),

  revokeTrustedDevices: () =>
    client.delete('/api/v1/auth/trusted-devices'),
}

export default client
