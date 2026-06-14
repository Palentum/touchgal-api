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
type AuthStatus = 'unknown' | 'authenticated' | 'unauthenticated' | 'account_disabled' | 'fetch_error'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<CurrentUser | null>(null)
  const loaded = ref(false)
  const loading = ref(false)
  const accountDisabled = ref(false)
  const authStatus = ref<AuthStatus>('unknown')
  const authErrorCode = ref<string | null>(null)
  const applicationAccess = useApplicationAccess()
  const nuxtApp = useNuxtApp()
  let fetchMeRequest: Promise<void> | null = null

  const applyMeResponse = (res: ApiResponse<CurrentUser> | null | undefined) => {
    if (res?.success) {
      user.value = res.data
      accountDisabled.value = res.data.status === 'disabled'
      authStatus.value = accountDisabled.value ? 'account_disabled' : 'authenticated'
      authErrorCode.value = null
      if (accountDisabled.value) {
        applicationAccess.resetApplications()
      }
      return
    }

    const code = res?.error.code || 'FETCH_ERROR'
    user.value = null
    accountDisabled.value = code === 'ACCOUNT_DISABLED'
    authStatus.value = accountDisabled.value ? 'account_disabled' : code === 'UNAUTHORIZED' ? 'unauthenticated' : 'fetch_error'
    authErrorCode.value = code
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
        const cachedFetchError = cached.value?.success === false && cached.value.error.code === 'FETCH_ERROR'
        const shouldRefresh = hasCached && (options.refresh || (import.meta.client && cachedFetchError))
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
          applyMeResponse({ success: false, error: { code: apiErrorCode(err) || 'FETCH_ERROR', message: '认证服务暂时不可用' } })
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
    authStatus.value = 'unauthenticated'
    authErrorCode.value = null
    loaded.value = true
    applicationAccess.resetApplications()
  }
  const isAccountDisabled = computed(() => accountDisabled.value || user.value?.status === 'disabled')

  const markAccountDisabled = () => {
    accountDisabled.value = true
    authStatus.value = 'account_disabled'
    authErrorCode.value = 'ACCOUNT_DISABLED'
    loaded.value = true
    applicationAccess.resetApplications()
  }

  const clearAccountDisabled = () => {
    accountDisabled.value = false
    if (authStatus.value === 'account_disabled') {
      authStatus.value = user.value ? 'authenticated' : 'unknown'
      authErrorCode.value = null
    }
  }

  const apiErrorCode = (err: unknown) => {
    return (err as { data?: { error?: { code?: string } } }).data?.error?.code
  }

  const hasAuthFetchError = computed(() => authStatus.value === 'fetch_error')

  return { user, loaded, loading, accountDisabled, authStatus, authErrorCode, isAccountDisabled, hasAuthFetchError, fetchMe, logout, markAccountDisabled, clearAccountDisabled }
})
