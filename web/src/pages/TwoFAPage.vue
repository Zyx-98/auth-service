<template>
  <div class="twofa-container">
    <div class="twofa-box">
      <h1>Two-Factor Authentication</h1>
      <p class="subtitle">Enter the 6-digit code from your authenticator app</p>

      <form @submit.prevent="handleVerify">
        <div class="form-group">
          <label for="code">Authentication Code:</label>
          <input
            id="code"
            v-model="form.code"
            type="text"
            inputmode="numeric"
            placeholder="000000"
            maxlength="6"
            pattern="[0-9]{6}"
            required
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

        <button type="submit" :disabled="loading || form.code.length !== 6">
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
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/auth'
import { clearAuthCookies } from '../utils/cookie'

const router = useRouter()
const loading = ref(false)
const error = ref('')

const form = ref({
  code: '',
  trustDevice: false,
})

onMounted(() => {
  const tempToken = sessionStorage.getItem('temp_token')
  if (!tempToken) {
    router.push('/login')
  }
})

const handleVerify = async () => {
  error.value = ''
  loading.value = true

  try {
    const response = await authApi.verifyTwoFALogin(form.value.code, form.value.trustDevice)
    const { data } = response.data

    if (!data || !data.token || !data.token.access_token) {
      throw new Error('No access token in response')
    }

    sessionStorage.removeItem('temp_token')
    sessionStorage.removeItem('user_email')

    // Wait longer for cookies to be processed, then do a hard navigation
    setTimeout(() => {
      window.location.href = '/dashboard'
    }, 500)
  } catch (err: any) {
    error.value = err.response?.data?.message || 'Invalid code. Please try again.'
    form.value.code = ''
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
