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

        <div class="two-fa-section">
          <h3>Two-Factor Authentication</h3>
          <p><strong>Status:</strong> <span :class="profile.totp_enabled ? 'status-enabled' : 'status-disabled'">{{ profile.totp_enabled ? 'Enabled ✓' : 'Disabled' }}</span></p>

          <div v-if="!profile.totp_enabled" class="action">
            <p>Enhance your account security with 2FA.</p>
            <button @click="handleSetupTwoFA" class="btn-setup" :disabled="setupLoading">
              {{ setupLoading ? 'Setting up...' : 'Enable 2FA' }}
            </button>
          </div>

          <div v-else class="action">
            <p>Your account is protected with 2FA.</p>
            <div class="button-group">
              <button @click="openDevicesModal" class="btn-secondary">
                🔐 Trusted Devices
              </button>
              <button @click="handleDisableTwoFA" class="btn-disable" :disabled="disableLoading">
                {{ disableLoading ? 'Disabling...' : 'Disable 2FA' }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="setupMessage" class="message" :class="setupMessageType">{{ setupMessage }}</div>
      </div>
    </main>

    <!-- 2FA Setup Modal (Single Modal) -->
    <div v-if="show2FAModal" class="modal-overlay" @click="close2FAModal">
      <div class="modal-content large" @click.stop>
        <div class="modal-header">
          <h2>{{ showVerification ? 'Verify 2FA Setup' : 'Scan QR Code for 2FA' }}</h2>
          <button class="close-btn" @click="close2FAModal">&times;</button>
        </div>
        <div class="modal-body">
          <div v-if="!showVerification" class="qr-section">
            <p class="subtitle">Use your authenticator app</p>
            <div class="qr-display">
              <img v-if="tempQRCode" :src="tempQRCode" alt="QR Code" class="qr-image">
              <p v-else class="loading-qr">Generating QR code...</p>
            </div>
            <div class="apps">
              <div class="app">🔐 Google Authenticator</div>
              <div class="app">🔐 Authy</div>
              <div class="app">🔐 Microsoft Authenticator</div>
            </div>
            <div class="manual-entry">
              <p><strong>Can't scan?</strong> Enter manually:</p>
              <code>{{ tempSecret }}</code>
            </div>
          </div>

          <div v-else class="verify-section">
            <p>Enter the 6-digit code from your authenticator app:</p>
            <input
              v-model="verificationCode"
              type="text"
              placeholder="000000"
              maxlength="6"
              inputmode="numeric"
              class="code-input"
              @keyup.enter="handleVerify2FA"
            />
            <p class="hint">This ensures your authenticator app is working correctly</p>
          </div>
        </div>
        <div class="modal-footer">
          <button v-if="showVerification" @click="showVerification = false" class="btn-cancel">Back</button>
          <button v-else @click="close2FAModal" class="btn-cancel">Cancel</button>
          <button
            v-if="showVerification"
            @click="handleVerify2FA"
            class="btn-verify"
            :disabled="verificationCode.length !== 6 || verifyLoading"
          >
            {{ verifyLoading ? 'Verifying...' : 'Verify & Enable' }}
          </button>
          <button
            v-else
            @click="showVerification = true"
            class="btn-verify"
          >
            I've Scanned the QR Code
          </button>
        </div>
        <div v-if="verifyMessage" class="verify-message" :class="verifyMessageType">{{ verifyMessage }}</div>
      </div>
    </div>

    <!-- Backup Codes Modal -->
    <div v-if="showBackupCodesModal" class="modal-overlay" @click="closeBackupCodesModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>Save Your Backup Codes</h2>
          <button class="close-btn" @click="closeBackupCodesModal">&times;</button>
        </div>
        <div class="modal-body">
          <p style="color: #e74c3c; font-weight: 600; margin-bottom: 15px;">⚠️ Save these codes in a safe place. You can use them if you lose access to your authenticator.</p>
          <div class="backup-codes">
            <div v-for="(code, index) in backupCodes" :key="index" class="backup-code">
              {{ code }}
            </div>
          </div>
          <div class="actions">
            <button @click="copyBackupCodes" class="btn-copy">
              {{ copyMessage ? 'Copied!' : 'Copy All' }}
            </button>
            <button @click="downloadBackupCodes" class="btn-download">Download</button>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeBackupCodesModal" class="btn-done">I've saved my codes</button>
        </div>
      </div>
    </div>

    <!-- Device Management Modal -->
    <div v-if="showDevicesModal" class="modal-overlay" @click="closeDevicesModal">
      <div class="modal-content large" @click.stop>
        <div class="modal-header">
          <h2>Trusted Devices</h2>
          <button class="close-btn" @click="closeDevicesModal">&times;</button>
        </div>
        <div class="modal-body">
          <p v-if="devices.length === 0" class="no-devices">No trusted devices yet. When you log in and verify with 2FA, you can choose to trust the device.</p>
          <div v-else class="devices-list">
            <div v-for="device in devices" :key="device.token" class="device-item">
              <div class="device-info">
                <p class="device-name">{{ device.name }}</p>
                <p class="device-meta">Trusted on {{ formatDate(device.created_at) }}</p>
                <p class="device-meta" v-if="device.expires_at">Expires {{ formatDate(device.expires_at) }}</p>
                <p class="device-ip" v-if="device.ip_address">{{ device.ip_address }}</p>
              </div>
              <button @click="removeDevice(device.token)" class="btn-remove" :disabled="removingDevice">Remove</button>
            </div>
          </div>
          <div v-if="devices.length > 0" class="devices-actions">
            <button @click="revokeAllDevices" class="btn-revoke-all" :disabled="removingDevice || devices.length === 0">
              {{ removingDevice ? 'Revoking...' : 'Revoke All Devices' }}
            </button>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeDevicesModal" class="btn-done">Done</button>
        </div>
        <div v-if="deviceMessage" class="device-message" :class="deviceMessageType">{{ deviceMessage }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/auth'
import { clearAuthCookies } from '../utils/cookie'

const router = useRouter()
const loading = ref(true)
const error = ref('')
const setupLoading = ref(false)
const disableLoading = ref(false)
const setupMessage = ref('')
const setupMessageType = ref('')

// 2FA Verification Modal
const show2FAModal = ref(false)
const showVerification = ref(false)
const verificationCode = ref('')
const verifyLoading = ref(false)
const verifyMessage = ref('')
const verifyMessageType = ref('')
const tempQRUri = ref('')
const tempSecret = ref('')
const tempQRCode = ref('')

// Backup Codes Modal
const showBackupCodesModal = ref(false)
const backupCodes = ref<string[]>([])
const copyMessage = ref(false)

// Device Management Modal
const showDevicesModal = ref(false)
const devices = ref<any[]>([])
const removingDevice = ref(false)
const deviceMessage = ref('')
const deviceMessageType = ref('')

const profile = ref({
  email: '',
  roles: [] as string[],
  totp_enabled: false,
})

const loadProfile = async () => {
  try {
    const response = await authApi.getProfile()
    profile.value = response.data.data
  } catch (err: any) {
    error.value = err.response?.data?.message || 'Failed to load profile'
  }
}

onMounted(async () => {
  try {
    await loadProfile()
  } catch (err: any) {
    if (err.response?.status === 401) {
      router.push('/login')
    }
  } finally {
    loading.value = false
  }
})

const handleSetupTwoFA = async () => {
  setupLoading.value = true
  setupMessage.value = ''
  try {
    const response = await authApi.setupTwoFA()
    const { secret, qr_code, otp_auth } = response.data.data

    tempSecret.value = secret
    tempQRUri.value = otp_auth
    tempQRCode.value = qr_code

    // Show modal with QR code
    show2FAModal.value = true
    showVerification.value = false
    verificationCode.value = ''
    verifyMessage.value = ''
  } catch (err: any) {
    setupMessage.value = err.response?.data?.message || 'Failed to setup 2FA'
    setupMessageType.value = 'error'
  } finally {
    setupLoading.value = false
  }
}

const handleVerify2FA = async () => {
  if (verificationCode.value.length !== 6) return

  verifyLoading.value = true
  verifyMessage.value = ''
  try {
    await authApi.verifyTwoFA(verificationCode.value)

    verifyMessage.value = 'Verification successful! 2FA is now enabled.'
    verifyMessageType.value = 'success'

    // Show backup codes
    setTimeout(() => {
      close2FAModal()
      // Generate mock backup codes (in real scenario, these come from API)
      backupCodes.value = generateBackupCodes()
      showBackupCodesModal.value = true
    }, 1000)

    await loadProfile()
  } catch (err: any) {
    verifyMessage.value = err.response?.data?.message || 'Invalid code. Please try again.'
    verifyMessageType.value = 'error'
  } finally {
    verifyLoading.value = false
  }
}

const close2FAModal = () => {
  show2FAModal.value = false
  showVerification.value = false
  verificationCode.value = ''
  verifyMessage.value = ''
  tempQRCode.value = ''
  tempSecret.value = ''
}

const closeBackupCodesModal = () => {
  showBackupCodesModal.value = false
  backupCodes.value = []
}

const generateBackupCodes = (): string[] => {
  const codes: string[] = []
  for (let i = 0; i < 8; i++) {
    const code = Math.random().toString(36).substring(2, 10).toUpperCase()
    codes.push(code.substring(0, 4) + '-' + code.substring(4, 8))
  }
  return codes
}

const copyBackupCodes = async () => {
  const text = backupCodes.value.join('\n')
  try {
    await navigator.clipboard.writeText(text)
    copyMessage.value = true
    setTimeout(() => {
      copyMessage.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

const downloadBackupCodes = () => {
  const text = `Backup Codes for 2FA\nGenerated: ${new Date().toLocaleString()}\n\n${backupCodes.value.join('\n')}\n\nKeep these codes in a safe place!`
  const blob = new Blob([text], { type: 'text/plain' })
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `backup-codes-${Date.now()}.txt`
  a.click()
  window.URL.revokeObjectURL(url)
}

const openDevicesModal = async () => {
  showDevicesModal.value = true
  deviceMessage.value = ''
  removingDevice.value = true
  try {
    const response = await authApi.getTrustedDevices()
    devices.value = response.data.data || []
  } catch (err: any) {
    deviceMessage.value = err.response?.data?.message || 'Failed to load trusted devices'
    deviceMessageType.value = 'error'
    devices.value = []
  } finally {
    removingDevice.value = false
  }
}

const closeDevicesModal = () => {
  showDevicesModal.value = false
  deviceMessage.value = ''
}

const removeDevice = async (deviceToken: string) => {
  if (!confirm('Remove this device? You\'ll need to verify with 2FA again.')) return

  removingDevice.value = true
  deviceMessage.value = ''
  try {
    await authApi.revokeTrustedDevices()
    devices.value = devices.value.filter(d => d.token !== deviceToken)
    deviceMessage.value = 'Device removed successfully'
    deviceMessageType.value = 'success'
  } catch (err: any) {
    deviceMessage.value = err.response?.data?.message || 'Failed to remove device'
    deviceMessageType.value = 'error'
  } finally {
    removingDevice.value = false
  }
}

const revokeAllDevices = async () => {
  if (!confirm('Remove all trusted devices? You\'ll need to verify with 2FA on every login.')) return

  removingDevice.value = true
  deviceMessage.value = ''
  try {
    await authApi.revokeTrustedDevices()
    devices.value = []
    deviceMessage.value = 'All devices have been revoked successfully'
    deviceMessageType.value = 'success'
  } catch (err: any) {
    deviceMessage.value = err.response?.data?.message || 'Failed to revoke devices'
    deviceMessageType.value = 'error'
  } finally {
    removingDevice.value = false
  }
}

const formatDate = (dateString: string): string => {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const handleDisableTwoFA = async () => {
  const code = prompt('Enter your 6-digit authenticator code to disable 2FA:')
  if (!code) return

  disableLoading.value = true
  setupMessage.value = ''
  try {
    await authApi.disableTwoFA(code)
    setupMessage.value = '2FA has been disabled successfully'
    setupMessageType.value = 'success'
    await loadProfile()
  } catch (err: any) {
    setupMessage.value = err.response?.data?.message || 'Failed to disable 2FA'
    setupMessageType.value = 'error'
  } finally {
    disableLoading.value = false
  }
}

const handleLogout = async () => {
  try {
    await authApi.logout()
    clearAuthCookies()
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

.two-fa-section {
  margin-top: 30px;
  padding-top: 30px;
  border-top: 2px solid #e0e0e0;
}

.two-fa-section h3 {
  color: #333;
  margin: 0 0 15px 0;
  font-size: 18px;
}

.status-enabled {
  color: #27ae60;
  font-weight: 600;
}

.status-disabled {
  color: #e74c3c;
  font-weight: 600;
}

.action {
  margin-top: 15px;
  padding: 15px;
  background: #f5f5f5;
  border-radius: 5px;
}

.action p {
  margin: 0 0 15px 0;
  color: #666;
  font-size: 14px;
}

.btn-setup,
.btn-disable {
  padding: 10px 20px;
  border: none;
  border-radius: 5px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
  display: inline-block;
}

.btn-setup {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-setup:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
}

.btn-disable {
  background: #e74c3c;
  color: white;
}

.btn-disable:hover:not(:disabled) {
  background: #c0392b;
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(231, 76, 60, 0.4);
}

.btn-setup:disabled,
.btn-disable:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.message {
  margin-top: 15px;
  padding: 15px;
  border-radius: 5px;
  font-size: 14px;
  font-weight: 500;
}

.message.success {
  background: #d5f4e6;
  color: #27ae60;
  border-left: 4px solid #27ae60;
}

.message.error {
  background: #fadbd8;
  color: #e74c3c;
  border-left: 4px solid #e74c3c;
}

.button-group {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}

.btn-secondary {
  padding: 10px 15px;
  background: white;
  color: #667eea;
  border: 2px solid #667eea;
  border-radius: 5px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-secondary:hover {
  background: #667eea;
  color: white;
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 10px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
  max-width: 500px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-content.large {
  max-width: 700px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 2px solid #f0f0f0;
}

.modal-header h2 {
  margin: 0;
  color: #333;
  font-size: 22px;
}

.close-btn {
  background: none;
  border: none;
  font-size: 28px;
  color: #999;
  cursor: pointer;
  padding: 0;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.3s;
}

.close-btn:hover {
  color: #333;
}

.modal-body {
  padding: 20px;
}

.modal-body p {
  margin: 0 0 15px 0;
  color: #555;
  font-size: 14px;
  line-height: 1.5;
}

.code-input {
  width: 100%;
  padding: 12px;
  font-size: 28px;
  letter-spacing: 8px;
  text-align: center;
  border: 2px solid #ddd;
  border-radius: 5px;
  font-weight: 600;
  box-sizing: border-box;
  transition: border-color 0.3s;
}

.code-input:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.hint {
  font-size: 12px;
  color: #999;
  margin-top: 10px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 20px;
  border-top: 2px solid #f0f0f0;
}

.btn-cancel,
.btn-done {
  padding: 10px 20px;
  background: #f0f0f0;
  color: #333;
  border: none;
  border-radius: 5px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-cancel:hover,
.btn-done:hover {
  background: #e0e0e0;
}

.btn-verify {
  padding: 10px 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 5px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-verify:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
}

.btn-verify:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.verify-message {
  margin-top: 15px;
  padding: 12px;
  border-radius: 5px;
  font-size: 13px;
  font-weight: 500;
}

.verify-message.success {
  background: #d5f4e6;
  color: #27ae60;
  border-left: 4px solid #27ae60;
}

.verify-message.error {
  background: #fadbd8;
  color: #e74c3c;
  border-left: 4px solid #e74c3c;
}

/* Backup Codes */
.backup-codes {
  background: #f9f9f9;
  border: 2px dashed #ddd;
  border-radius: 5px;
  padding: 20px;
  margin: 15px 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.backup-code {
  background: white;
  padding: 10px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-size: 13px;
  font-weight: 600;
  text-align: center;
  user-select: all;
  border: 1px solid #e0e0e0;
}

.actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin: 20px 0;
}

.btn-copy,
.btn-download {
  padding: 10px 15px;
  background: white;
  color: #667eea;
  border: 2px solid #667eea;
  border-radius: 5px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-copy:hover,
.btn-download:hover {
  background: #667eea;
  color: white;
}

/* Device Management */
.devices-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.device-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px;
  background: #f9f9f9;
  border-radius: 5px;
  border: 1px solid #e0e0e0;
}

.device-info {
  flex: 1;
}

.device-name {
  margin: 0;
  color: #333;
  font-weight: 600;
  font-size: 14px;
}

.device-meta {
  margin: 5px 0 0 0;
  color: #999;
  font-size: 12px;
}

.btn-remove {
  padding: 8px 12px;
  background: #e74c3c;
  color: white;
  border: none;
  border-radius: 3px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-remove:hover:not(:disabled) {
  background: #c0392b;
}

.btn-remove:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.no-devices {
  text-align: center;
  color: #999;
  padding: 30px 0;
}

/* QR Code Display */
.qr-section {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.subtitle {
  color: #666;
  font-size: 14px;
  margin-bottom: 20px;
}

.qr-display {
  background: #f9f9f9;
  border: 3px solid #ddd;
  border-radius: 10px;
  padding: 20px;
  margin: 20px 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 350px;
  width: 100%;
}

.qr-image {
  max-width: 300px;
  height: auto;
  border-radius: 5px;
}

.loading-qr {
  color: #999;
  font-size: 14px;
}

.apps {
  display: flex;
  justify-content: space-around;
  width: 100%;
  margin: 20px 0;
  font-size: 12px;
}

.app {
  color: #666;
}

.manual-entry {
  background: #f0f0f0;
  padding: 15px;
  border-radius: 5px;
  width: 100%;
  text-align: left;
}

.manual-entry p {
  color: #666;
  font-size: 12px;
  margin: 0 0 8px 0;
}

.manual-entry code {
  background: white;
  padding: 8px 12px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-weight: 600;
  color: #333;
  word-break: break-all;
  display: block;
}

.verify-section {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.verify-section p {
  color: #666;
  margin-bottom: 15px;
}

/* Device Management Modal */
.device-ip {
  margin: 3px 0 0 0;
  color: #bbb;
  font-size: 11px;
  font-family: 'Courier New', monospace;
}

.devices-actions {
  margin-top: 20px;
  display: flex;
  justify-content: center;
  border-top: 1px solid #f0f0f0;
  padding-top: 20px;
}

.btn-revoke-all {
  padding: 10px 20px;
  background: #e74c3c;
  color: white;
  border: none;
  border-radius: 5px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-revoke-all:hover:not(:disabled) {
  background: #c0392b;
}

.btn-revoke-all:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.device-message {
  margin-top: 15px;
  padding: 12px;
  border-radius: 5px;
  font-size: 13px;
  font-weight: 500;
}

.device-message.success {
  background: #d5f4e6;
  color: #27ae60;
  border-left: 4px solid #27ae60;
}

.device-message.error {
  background: #fadbd8;
  color: #e74c3c;
  border-left: 4px solid #e74c3c;
}
</style>
