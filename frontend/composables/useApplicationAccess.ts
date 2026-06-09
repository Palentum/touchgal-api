import type { ApplicationItem } from '~/composables/useDashboard'

const approvedApplicationStatus = 'approved'

export const useApplicationAccess = () => {
  const applications = useState<ApplicationItem[]>('dashboard:applications', () => [])
  const loaded = useState('dashboard:applicationsLoaded', () => false)
  const loading = useState('dashboard:applicationsLoading', () => false)
  const checked = useState('dashboard:applicationsChecked', () => false)
  const userId = useState<string | null>('dashboard:applicationsUserId', () => null)

  const setUserScope = (currentUserId?: string) => {
    if (!currentUserId || userId.value === currentUserId) {
      return
    }
    applications.value = []
    loaded.value = false
    checked.value = false
    userId.value = currentUserId
  }

  const hasApprovedApplication = computed(() => applications.value.some((app) => app.status === approvedApplicationStatus))

  const fetchApplications = async (currentUserId?: string, force = false) => {
    setUserScope(currentUserId)
    if (loaded.value && !force) {
      checked.value = true
      return true
    }

    loading.value = true
    checked.value = false
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
      checked.value = true
      loading.value = false
    }
  }

  const resetApplications = () => {
    applications.value = []
    loaded.value = false
    checked.value = false
    loading.value = false
    userId.value = null
  }

  return { applications, loaded, checked, loading, hasApprovedApplication, fetchApplications, resetApplications }
}
