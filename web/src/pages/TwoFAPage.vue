<template>
  <div class="twofa-container">
    <div class="twofa-box">
      <h1>Two-Factor Authentication</h1>

      <form @submit.prevent="handleVerify">
        <div class="form-group">
          <label for="code">Authentication Code or Backup Code:</label>
          <input
            id="code"
            v-model="form.code"
            type="text"
            placeholder="XXXXXX"
            required
            autofocus
            @input="normalizeCode"
          />
        </div>

        <div class="form-group checkbox-group">
          <input
            id="trust"
            v-model="form.trustDevice"
            type="checkbox"
          />
          <label for="trust" class="checkbox-label">Trust this device for 30 days</label>
        </div>

        <button type="submit" :disabled="loading || !canSubmit">
          {{ loading ? 'Verifying...' : 'Verify' }}
        </button>

        <p class="error" v-if="error">{{ error }}</p>
      </form>

      <p class="back-link">
        <router-link to="/login">Back to login</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/auth'

const router = useRouter()
const loading = ref(false)
const error = ref('')

const form = ref({
  code: '',
  trustDevice: false,
})

const canSubmit = computed(() => {
  const codeWithoutHyphen = form.value.code.replace('-', '')
  const isTotp = /^[0-9]{6}$/.test(codeWithoutHyphen)
  const isBackupCode = /^[A-Z0-9]{6,8}$/.test(codeWithoutHyphen)
  return isTotp || isBackupCode
})

onMounted(() => {
  const tempToken = sessionStorage.getItem('temp_token')
  const totpToken = sessionStorage.getItem('totp_token')
  if (!tempToken && !totpToken) {
    router.push('/login')
  }
})

const normalizeCode = () => {
  const input = form.value.code.toUpperCase()
  form.value.code = input.replace(/[^A-Z0-9\-]/g, '').slice(0, 8)
}

const handleVerify = async () => {
  error.value = ''
  loading.value = true

  try {
    const totpToken = sessionStorage.getItem('totp_token')
    const response = totpToken
      ? await authApi.verifyOAuthTOTP(form.value.code, totpToken, form.value.trustDevice)
      : await authApi.verifyTwoFALogin(form.value.code, form.value.trustDevice)
    const { data } = response.data

    if (!data?.access_token && !data?.token?.access_token) {
      throw new Error('No access token in response')
    }

    sessionStorage.removeItem('temp_token')
    sessionStorage.removeItem('user_email')
    sessionStorage.removeItem('totp_token')
    sessionStorage.removeItem('is_new_user')

    setTimeout(() => {
      window.location.href = '/dashboard'
    }, 500)
  } catch (err: any) {
    const errorCode = err.response?.data?.code
    const message = err.response?.data?.message

    if (errorCode === 'expired_2fa_session') {
      error.value = 'Session expired. Please log in again.'
      setTimeout(() => {
        router.push('/login')
      }, 1500)
    } else if (errorCode === 'invalid_2fa_code') {
      error.value = message || 'Invalid code. Please try again.'
      form.value.code = ''
    } else {
      error.value = message || 'Invalid code. Please try again.'
      form.value.code = ''
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.twofa-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu,
    Cantarell, sans-serif;
}

.twofa-box {
  background: white;
  border-radius: 10px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
  padding: 40px;
  width: 100%;
  max-width: 400px;
}

h1 {
  text-align: center;
  color: #333;
  margin-bottom: 10px;
  font-size: 28px;
}

.subtitle {
  text-align: center;
  color: #666;
  margin-bottom: 30px;
  font-size: 14px;
}

.form-group {
  margin-bottom: 20px;
}

.checkbox-group {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
}

.checkbox-group input[type='checkbox'] {
  width: 18px;
  height: 18px;
  cursor: pointer;
  margin: 0;
}

.checkbox-label {
  display: inline;
  margin: 0;
  font-size: 14px;
  cursor: pointer;
  user-select: none;
}

label {
  display: block;
  margin-bottom: 8px;
  color: #555;
  font-weight: 500;
}

input[type='text'] {
  width: 100%;
  padding: 12px;
  border: 1px solid #ddd;
  border-radius: 5px;
  font-size: 20px;
  text-align: center;
  letter-spacing: 4px;
  font-family: 'Courier New', monospace;
  box-sizing: border-box;
  transition: border-color 0.3s;
}

input[type='text']:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

button[type='submit'] {
  width: 100%;
  padding: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 5px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
  margin-bottom: 15px;
}

button[type='submit']:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
}

button[type='submit']:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.error {
  color: #e74c3c;
  text-align: center;
  margin-top: 15px;
  font-size: 14px;
}

.back-link {
  text-align: center;
  margin-top: 20px;
  color: #666;
  font-size: 14px;
}

.back-link a {
  color: #667eea;
  text-decoration: none;
  font-weight: 600;
}

.back-link a:hover {
  text-decoration: underline;
}
</style>
