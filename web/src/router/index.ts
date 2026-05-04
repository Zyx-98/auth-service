import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import LoginPage from '../pages/LoginPage.vue'
import RegisterPage from '../pages/RegisterPage.vue'
import TwoFAPage from '../pages/TwoFAPage.vue'
import DashboardPage from '../pages/DashboardPage.vue'
import { authApi } from '../api/auth'

const isAuthenticated = async (): Promise<boolean> => {
  try {
    await authApi.getProfile({
      skipAuthRefresh: true,
      skipAuthRedirect: true,
    })
    return true
  } catch {
    return false
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/login',
    name: 'Login',
    component: LoginPage,
    meta: { guestOnly: true },
  },
  {
    path: '/register',
    name: 'Register',
    component: RegisterPage,
    meta: { guestOnly: true },
  },
  {
    path: '/2fa',
    name: 'TwoFA',
    component: TwoFAPage,
    meta: { guestOnly: true, requiresTwoFAToken: true },
  },
  {
    path: '/2fa-verify',
    redirect: '/2fa',
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: DashboardPage,
    meta: { requiresAuth: true },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const loggedIn = await isAuthenticated()

  if (to.meta.requiresAuth && !loggedIn) {
    return {
      path: '/login',
      query: { redirect: to.fullPath },
    }
  }

  if (to.meta.guestOnly && loggedIn) {
    return '/dashboard'
  }

  if (
    to.meta.requiresTwoFAToken &&
    !sessionStorage.getItem('temp_token') &&
    !sessionStorage.getItem('totp_token')
  ) {
    return '/login'
  }
})

export default router
