import { defineStore } from 'pinia'

export interface CurrentUser {
  id: string
  email: string
  displayName: string
  status: string
  isAdmin: boolean
  createdAt: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<CurrentUser | null>(null)
  const loaded = ref(false)
  const loading = ref(false)

  const fetchMe = async () => {
    loading.value = true
    try {
      const { apiFetch } = useApi()
      const res = await apiFetch<CurrentUser>('/auth/me')
      user.value = res.success ? res.data : null
    } catch {
      user.value = null
    } finally {
      loaded.value = true
      loading.value = false
    }
  }

  const logout = async () => {
    const { apiFetch } = useApi()
    await apiFetch('/auth/logout', { method: 'POST' })
    user.value = null
    loaded.value = true
  }

  return { user, loaded, loading, fetchMe, logout }
})
