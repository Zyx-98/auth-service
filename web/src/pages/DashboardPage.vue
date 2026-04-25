<template>
  <div class="dashboard-container">
    <header class="header">
      <h1>Dashboard</h1>
      <button @click="handleLogout" class="logout-btn">Logout</button>
    </header>

    <main class="content">
      <div v-if="loading" class="loading">Loading...</div>
      <div v-else-if="error" class="error">{{ error }}</div>
      <div v-else class="profile-card">
        <h2>Profile</h2>
        <p><strong>Email:</strong> {{ profile.email }}</p>
        <p><strong>Roles:</strong> {{ profile.roles.join(', ') || 'None' }}</p>
        <p><strong>2FA Enabled:</strong> {{ profile.totp_enabled ? 'Yes' : 'No' }}</p>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/auth'

const router = useRouter()
const loading = ref(true)
const error = ref('')

const profile = ref({
  email: '',
  roles: [] as string[],
  totp_enabled: false,
})

onMounted(async () => {
  try {
    const response = await authApi.getProfile()
    profile.value = response.data.data
  } catch (err: any) {
    error.value = err.response?.data?.message || 'Failed to load profile'
  } finally {
    loading.value = false
  }
})

const handleLogout = async () => {
  try {
    await authApi.logout()
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    router.push('/login')
  } catch (err) {
    error.value = 'Failed to logout'
  }
}
</script>

<style scoped>
.dashboard-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu,
    Cantarell, sans-serif;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 30px 50px;
  background: rgba(0, 0, 0, 0.1);
  color: white;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

h1 {
  margin: 0;
  font-size: 32px;
}

.logout-btn {
  padding: 10px 20px;
  background: rgba(255, 255, 255, 0.2);
  color: white;
  border: 1px solid white;
  border-radius: 5px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.3s;
}

.logout-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.content {
  padding: 40px;
  max-width: 800px;
  margin: 0 auto;
}

.loading,
.error {
  background: white;
  padding: 20px;
  border-radius: 10px;
  text-align: center;
}

.error {
  color: #e74c3c;
}

.profile-card {
  background: white;
  padding: 30px;
  border-radius: 10px;
  box-shadow: 0 5px 15px rgba(0, 0, 0, 0.1);
}

.profile-card h2 {
  margin-top: 0;
  color: #333;
  border-bottom: 2px solid #667eea;
  padding-bottom: 15px;
}

.profile-card p {
  margin: 15px 0;
  color: #555;
  font-size: 16px;
}
</style>
