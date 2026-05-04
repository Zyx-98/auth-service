export function getCookie(name: string): string | null {
  const value = `; ${document.cookie}`
  const parts = value.split(`; ${name}=`)
  if (parts.length === 2) {
    return parts.pop()?.split(';').shift() || null
  }
  return null
}

export function getAccessToken(): string | null {
  return getCookie('access_token')
}

export function getRefreshToken(): string | null {
  return getCookie('refresh_token')
}

export function getDeviceToken(): string | null {
  return getCookie('device_token')
}

export function hasValidTokens(): boolean {
  return !!getAccessToken() && !!getRefreshToken()
}

export function clearAuthCookies(): void {
  document.cookie = 'access_token=; Max-Age=-1; path=/;'
  document.cookie = 'refresh_token=; Max-Age=-1; path=/;'
  document.cookie = 'device_token=; Max-Age=-1; path=/;'
}
