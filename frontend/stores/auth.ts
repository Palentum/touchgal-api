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
  const accountDisabled = ref(false)
  const applicationAccess = useApplicationAccess()

  const fetchMe = async () => {
    loading.value = true
    try {
      const { apiFetch } = useApi()
      const res = await apiFetch<CurrentUser>('/auth/me')
      if (res.success) {
        user.value = res.data
        accountDisabled.value = res.data.status === 'disabled'
        if (accountDisabled.value) {
          applicationAccess.resetApplications()
        }
      } else {
        user.value = null
        accountDisabled.value = res.error.code === 'ACCOUNT_DISABLED'
        applicationAccess.resetApplications()
      }
    } catch (err) {
      user.value = null
      accountDisabled.value = apiErrorCode(err) === 'ACCOUNT_DISABLED'
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
    accountDisabled.value = false
    loaded.value = true
    applicationAccess.resetApplications()
  }
  const isAccountDisabled = computed(() => accountDisabled.value || user.value?.status === 'disabled')

  const markAccountDisabled = () => {
    accountDisabled.value = true
    loaded.value = true
    applicationAccess.resetApplications()
  }

  const clearAccountDisabled = () => {
    accountDisabled.value = false
  }

  const apiErrorCode = (err: unknown) => {
    return (err as { data?: { error?: { code?: string } } }).data?.error?.code
  }

  return { user, loaded, loading, accountDisabled, isAccountDisabled, fetchMe, logout, markAccountDisabled, clearAccountDisabled }
})
