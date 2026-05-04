<template>
  <div class="callback-container">
    <div class="callback-box">
      <h1 id="title">Processing Login</h1>
      <div class="spinner"></div>
      <p id="message">Completing authentication...</p>
      <p class="status" id="status">⏳ Processing</p>
      <p class="error" v-if="error">{{ error }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { authApi } from '../api/auth'

const router = useRouter()
const route = useRoute()
const error = ref('')

onMounted(async () => {
  const code = route.query.code as string
  const state = route.query.state as string

  if (!code || !state) {
    error.value = 'Missing authentication parameters'
    setTimeout(() => router.push('/login'), 2000)
    return
  }

  try {
    const response = await authApi.googleCallback(code, state)
    const callbackResp = response.data.data

    console.log('OAuth callback response:', callbackResp)

    if (callbackResp.totp_required && callbackResp.totp_token) {
      document.getElementById('title')!.textContent = 'Two-Factor Authentication Required'
      document.getElementById('message')!.textContent = 'Redirecting to verify your authenticator...'
      document.getElementById('status')!.textContent = '🔐 Enter your 2FA code'

      sessionStorage.setItem('totp_token', callbackResp.totp_token)
      sessionStorage.setItem('is_new_user', callbackResp.is_new_user ? 'true' : 'false')

      setTimeout(() => {
        router.push('/2fa')
      }, 500)
    } else {
      document.getElementById('title')!.textContent = 'Login Successful'
      document.getElementById('message')!.textContent = 'Redirecting to dashboard...'
      document.getElementById('status')!.textContent = '✓ Authentication complete'

      setTimeout(() => {
        router.push('/dashboard')
      }, 1000)
    }
  } catch (err: any) {
    error.value = err.response?.data?.message || 'Authentication failed'
    console.error('OAuth callback error:', err)
    setTimeout(() => router.push('/login'), 2000)
  }
})
</script>

<style scoped>
.callback-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu,
    Cantarell, sans-serif;
}

.callback-box {
  background: white;
  border-radius: 10px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
  padding: 40px;
  max-width: 400px;
  text-align: center;
}

h1 {
  color: #333;
  margin-bottom: 10px;
  font-size: 24px;
}

p {
  color: #666;
  margin-bottom: 20px;
  font-size: 14px;
}

.spinner {
  border: 4px solid #f3f3f3;
  border-top: 4px solid #667eea;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
  margin: 0 auto 20px;
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.status {
  color: #27ae60;
  margin-top: 20px;
  font-size: 14px;
}

.error {
  color: #e74c3c;
  margin-top: 15px;
}
</style>
