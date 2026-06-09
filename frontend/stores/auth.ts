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
  const applicationAccess = useApplicationAccess()

  const fetchMe = async () => {
    loading.value = true
    try {
      const { apiFetch } = useApi()
      const res = await apiFetch<CurrentUser>('/auth/me')
      if (res.success) {
        user.value = res.data
      } else {
        user.value = null
        applicationAccess.resetApplications()
      }
    } catch {
      user.value = null
      applicationAccess.resetApplications()
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
    applicationAccess.resetApplications()
  }

  return { user, loaded, loading, fetchMe, logout }
})
