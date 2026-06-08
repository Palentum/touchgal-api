import type { ApplicationItem } from '~/composables/useDashboard'

const approvedApplicationStatus = 'approved'

export const useApplicationAccess = () => {
  const applications = useState<ApplicationItem[]>('dashboard:applications', () => [])
  const loaded = useState('dashboard:applicationsLoaded', () => false)
  const loading = useState('dashboard:applicationsLoading', () => false)
  const userId = useState<string | null>('dashboard:applicationsUserId', () => null)
  const adminApproved = useState('dashboard:applicationsAdminApproved', () => false)

  const setUserScope = (currentUserId?: string, currentUserIsAdmin = false) => {
    adminApproved.value = currentUserIsAdmin
    if (!currentUserId || userId.value === currentUserId) {
      return
    }
    applications.value = []
    loaded.value = false
    userId.value = currentUserId
  }

  const hasApprovedApplication = computed(() => adminApproved.value || applications.value.some((app) => app.status === approvedApplicationStatus))

  const fetchApplications = async (currentUserId?: string, force = false, currentUserIsAdmin = false) => {
    setUserScope(currentUserId, currentUserIsAdmin)
    if (currentUserIsAdmin) {
      loaded.value = true
      loading.value = false
      return true
    }
    if (loaded.value && !force) {
      return true
    }

    loading.value = true
    try {
      const { apiFetch } = useApi()
      const res = await apiFetch<ApplicationItem[]>('/applications')
      if (!res.success) {
        applications.value = []
        loaded.value = false
        return false
      }
      applications.value = res.data
      loaded.value = true
      return true
    } catch {
      applications.value = []
      loaded.value = false
      return false
    } finally {
      loading.value = false
    }
  }

  const resetApplications = () => {
    applications.value = []
    loaded.value = false
    loading.value = false
    userId.value = null
    adminApproved.value = false
  }

  return { applications, loaded, loading, hasApprovedApplication, fetchApplications, resetApplications }
}
