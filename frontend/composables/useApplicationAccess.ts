import type { ApplicationItem } from '~/composables/useDashboard'

const applicationsCacheTTL = 60 * 1000

let applicationsRequestUserId: string | null = null
let applicationsRequestId = 0
let applicationsRequest: Promise<boolean> | null = null

const approvedApplicationStatus = 'approved'
type EnsureApplicationsOptions = {
  staleWhileRevalidate?: boolean
}

export const useApplicationAccess = () => {
  const applications = useState<ApplicationItem[]>('dashboard:applications', () => [])
  const loaded = useState('dashboard:applicationsLoaded', () => false)
  const loading = useState('dashboard:applicationsLoading', () => false)
  const checked = useState('dashboard:applicationsChecked', () => false)
  const userId = useState<string | null>('dashboard:applicationsUserId', () => null)
  const fetchedAt = useState('dashboard:applicationsFetchedAt', () => 0)

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

  const refreshApplications = async (currentUserId?: string, background = false) => {
    if (!setUserScope(currentUserId)) {
      checked.value = true
      return false
    }

    const requestUserId = userId.value
    if (applicationsRequest && applicationsRequestUserId === requestUserId) {
      return applicationsRequest
    }

    loading.value = true
    if (!background) {
      checked.value = false
    }

    const requestId = ++applicationsRequestId
    const preserveStaleData = background && loaded.value
    const isCurrentRequest = () => userId.value === requestUserId && applicationsRequestId === requestId
    const request = (async () => {
      try {
        const { apiFetch } = useApi()
        const res = await apiFetch<ApplicationItem[]>('/applications')
        if (!isCurrentRequest()) {
          return false
        }
        if (!res.success) {
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
      } catch {
        if (isCurrentRequest()) {
          if (preserveStaleData) {
            fetchedAt.value = Date.now()
          } else {
            applications.value = []
            loaded.value = false
            fetchedAt.value = 0
          }
        }
        return false
      } finally {
        if (applicationsRequestId === requestId) {
          applicationsRequest = null
          applicationsRequestUserId = null
        }
        if (isCurrentRequest()) {
          checked.value = true
          loading.value = false
        }
      }
    })()

    applicationsRequest = request
    applicationsRequestUserId = requestUserId
    return request
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
    applications.value = []
    loaded.value = false
    checked.value = false
    loading.value = false
    fetchedAt.value = 0
    userId.value = null
    applicationsRequest = null
    applicationsRequestId++
    applicationsRequestUserId = null
  }

  return { applications, loaded, checked, loading, hasApprovedApplication, ensureApplications, refreshApplications, upsertApplication, invalidateApplications, resetApplications }
}
