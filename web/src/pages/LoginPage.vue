<template>
  <div class="login-container">
    <div class="login-box">
      <h1>Login</h1>

      <form @submit.prevent="handleLogin">
        <div class="form-group">
          <label for="email">Email:</label>
          <input
            id="email"
            v-model="form.email"
            type="email"
            placeholder="you@example.com"
            required
          />
        </div>

        <div class="form-group">
          <label for="password">Password:</label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            placeholder="Enter your password"
            required
          />
        </div>

        <button type="submit" :disabled="loading">
          {{ loading ? 'Logging in...' : 'Login' }}
        </button>

        <p class="error" v-if="error">{{ error }}</p>
      </form>

      <div class="divider">OR</div>

      <button class="google-btn" @click="handleGoogleLogin" :disabled="loading">
        Login with Google
      </button>

      <p class="register-link">
        Don't have an account?
        <router-link to="/register">Register here</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/auth'

const router = useRouter()
const loading = ref(false)
const error = ref('')

interface LoginForm {
  email: string
  password: string
}

const form = ref<LoginForm>({
  email: '',
  password: '',
})

const handleLogin = async () => {
  error.value = ''
  loading.value = true

  try {
    const response = await authApi.login(form.value.email, form.value.password)
    const { data } = response.data

    if (data.requires_2fa) {
      localStorage.setItem('temp_token', data.temp_token)
      localStorage.setItem('user_email', form.value.email)
      router.push('/2fa')
    } else if (data.token) {
      localStorage.setItem('access_token', data.token.access_token)
      localStorage.setItem('refresh_token', data.token.refresh_token)
      router.push('/dashboard')
    }
  } catch (err: any) {
    error.value = err.response?.data?.message || 'Login failed. Please try again.'
  } finally {
    loading.value = false
  }
}

const handleGoogleLogin = async () => {
  loading.value = true
  try {
    const response = await authApi.googleLoginRedirect()
    const { auth_url } = response.data
    window.location.href = auth_url
  } catch (err) {
    error.value = 'Failed to initiate Google login'
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu,
    Cantarell, sans-serif;
}

.login-box {
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
  margin-bottom: 30px;
  font-size: 28px;
}

.form-group {
  margin-bottom: 20px;
}

label {
  display: block;
  margin-bottom: 8px;
  color: #555;
  font-weight: 500;
}

input[type='email'],
input[type='password'] {
  width: 100%;
  padding: 12px;
  border: 1px solid #ddd;
  border-radius: 5px;
  font-size: 14px;
  box-sizing: border-box;
  transition: border-color 0.3s;
}

input[type='email']:focus,
input[type='password']:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

button {
  width: 100%;
  padding: 12px;
  border: none;
  border-radius: 5px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

button[type='submit'] {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  margin-bottom: 15px;
}

button[type='submit']:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
}

.google-btn {
  background: white;
  color: #333;
  border: 2px solid #ddd;
  margin-bottom: 20px;
}

.google-btn:hover:not(:disabled) {
  background: #f5f5f5;
  border-color: #667eea;
}

.divider {
  text-align: center;
  color: #999;
  margin: 20px 0;
  font-size: 14px;
}

.error {
  color: #e74c3c;
  text-align: center;
  margin-top: 15px;
  font-size: 14px;
}

.register-link {
  text-align: center;
  margin-top: 20px;
  color: #666;
  font-size: 14px;
}

.register-link a {
  color: #667eea;
  text-decoration: none;
  font-weight: 600;
}

.register-link a:hover {
  text-decoration: underline;
}
</style>
