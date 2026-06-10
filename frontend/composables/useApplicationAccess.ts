import type { ApiResponse } from '~/composables/useApi'
import type { ApplicationItem } from '~/composables/useDashboard'

const applicationsCacheTTL = 60 * 1000
const approvedApplicationStatus = 'approved'

type EnsureApplicationsOptions = {
  staleWhileRevalidate?: boolean
}

const applicationsDataKey = (currentUserId: string | null) => `dashboard:applications:${currentUserId || 'anonymous'}`

export const useApplicationAccess = () => {
  const applications = useState<ApplicationItem[]>('dashboard:applications', () => [])
  const loaded = useState('dashboard:applicationsLoaded', () => false)
  const loading = useState('dashboard:applicationsLoading', () => false)
  const checked = useState('dashboard:applicationsChecked', () => false)
  const userId = useState<string | null>('dashboard:applicationsUserId', () => null)
  const fetchedAt = useState('dashboard:applicationsFetchedAt', () => 0)
  const nuxtApp = useNuxtApp()

  const setUserScope = (currentUserId?: string) => {
    if (!currentUserId) {
      return userId.value !== null
    }
    if (userId.value === currentUserId) {
      return true
    }
    applications.value = []
    loaded.value = false
    checked.value = false
    fetchedAt.value = 0
    userId.value = currentUserId
    return true
  }

  const hasApprovedApplication = computed(() => applications.value.some((app) => app.status === approvedApplicationStatus))

  const isFresh = () => loaded.value && Date.now() - fetchedAt.value < applicationsCacheTTL

  const applyApplicationsResponse = (res: ApiResponse<ApplicationItem[]> | null | undefined, preserveStaleData: boolean) => {
    if (!res?.success) {
      if (preserveStaleData) {
        fetchedAt.value = Date.now()
      } else {
        applications.value = []
        loaded.value = false
        fetchedAt.value = 0
      }
      return false
    }

    applications.value = res.data
    loaded.value = true
    fetchedAt.value = Date.now()
    return true
  }

  const refreshApplications = async (currentUserId?: string, background = false) => {
    if (!setUserScope(currentUserId)) {
      checked.value = true
      return false
    }

    const requestUserId = userId.value
    const preserveStaleData = background && loaded.value
    loading.value = true
    if (!background) {
      checked.value = false
    }

    try {
      const key = applicationsDataKey(requestUserId)
      const cached = await nuxtApp.runWithContext(() => useNuxtData<ApiResponse<ApplicationItem[]>>(key).data)
      const hasCached = cached.value !== null && cached.value !== undefined
      const shouldRefresh = hasCached && (background || !loaded.value || !isFresh())
      const { data, refresh } = await nuxtApp.runWithContext(() => {
        const { apiData } = useApi()
        return apiData<ApplicationItem[]>(key, '/applications', { immediate: !shouldRefresh })
      })
      if (shouldRefresh) {
        await refresh()
      }
      if (userId.value !== requestUserId) {
        return false
      }
      return applyApplicationsResponse(data.value, preserveStaleData)
    } finally {
      if (userId.value === requestUserId) {
        checked.value = true
        loading.value = false
      }
    }
  }

  const ensureApplications = async (currentUserId?: string, options: EnsureApplicationsOptions = {}) => {
    if (!setUserScope(currentUserId)) {
      checked.value = true
      return false
    }
    if (isFresh()) {
      checked.value = true
      return true
    }
    if (loaded.value && options.staleWhileRevalidate) {
      checked.value = true
      void refreshApplications(currentUserId, true)
      return true
    }
    return await refreshApplications(currentUserId)
  }

  const upsertApplication = (application: ApplicationItem, currentUserId?: string) => {
    if (!setUserScope(currentUserId)) {
      return
    }
    applications.value = [application, ...applications.value.filter((app) => app.id !== application.id)]
    loaded.value = true
    checked.value = true
    fetchedAt.value = Date.now()
  }

  const invalidateApplications = () => {
    fetchedAt.value = 0
  }

  const resetApplications = () => {
    nuxtApp.runWithContext(() => clearNuxtData(applicationsDataKey(userId.value)))
    applications.value = []
    loaded.value = false
    checked.value = false
    loading.value = false
    fetchedAt.value = 0
    userId.value = null
  }

  return { applications, loaded, checked, loading, hasApprovedApplication, ensureApplications, refreshApplications, upsertApplication, invalidateApplications, resetApplications }
}
