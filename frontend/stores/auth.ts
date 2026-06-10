import { defineStore } from 'pinia'
import type { ApiResponse } from '~/composables/useApi'

export interface CurrentUser {
  id: string
  email: string
  displayName: string
  status: string
  isAdmin: boolean
  minuteLimit: number
  dailyLimit: number
  createdAt: string
}

type FetchMeOptions = {
  refresh?: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<CurrentUser | null>(null)
  const loaded = ref(false)
  const loading = ref(false)
  const accountDisabled = ref(false)
  const applicationAccess = useApplicationAccess()
  const nuxtApp = useNuxtApp()
  let fetchMeRequest: Promise<void> | null = null

  const applyMeResponse = (res: ApiResponse<CurrentUser> | null | undefined) => {
    if (res?.success) {
      user.value = res.data
      accountDisabled.value = res.data.status === 'disabled'
      if (accountDisabled.value) {
        applicationAccess.resetApplications()
      }
      return
    }

    user.value = null
    accountDisabled.value = res?.error.code === 'ACCOUNT_DISABLED'
    applicationAccess.resetApplications()
  }

  const fetchMe = async (options: FetchMeOptions = {}) => {
    if (fetchMeRequest) {
      return await fetchMeRequest
    }

    fetchMeRequest = nuxtApp.runWithContext(async () => {
      loading.value = true
      try {
        const { apiData } = useApi()
        const cached = useNuxtData<ApiResponse<CurrentUser>>('auth:me').data
        const hasCached = cached.value !== null && cached.value !== undefined
        const shouldRefresh = options.refresh && hasCached
        const { data, refresh } = await apiData<CurrentUser>('auth:me', '/auth/me', { immediate: !shouldRefresh })
        if (shouldRefresh) {
          await refresh()
        }
        applyMeResponse(data.value)
      } catch (err) {
        const data = (err as { data?: ApiResponse<CurrentUser> }).data
        if (data && data.success === false) {
          applyMeResponse(data)
        } else {
          user.value = null
          accountDisabled.value = apiErrorCode(err) === 'ACCOUNT_DISABLED'
          applicationAccess.resetApplications()
        }
      } finally {
        loaded.value = true
        loading.value = false
      }
    })

    try {
      await fetchMeRequest
    } finally {
      fetchMeRequest = null
    }
  }

  const logout = async () => {
    const { apiFetch } = useApi()
    await apiFetch('/auth/logout', { method: 'POST' })
    nuxtApp.runWithContext(() => clearNuxtData('auth:me'))
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
