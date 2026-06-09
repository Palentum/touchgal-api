export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) {
    return
  }
  const auth = useAuthStore()
  if (!auth.loaded) {
    await auth.fetchMe()
  }
  if (!auth.user) {
    return navigateTo(`/auth/login?redirect=${encodeURIComponent(to.fullPath)}`)
  }
  if (to.path.startsWith('/dashboard')) {
    const access = useApplicationAccess()
    const ok = await access.fetchApplications(auth.user.id, true)
    if (!ok) {
      if (to.path !== '/dashboard/apply') {
        return navigateTo('/dashboard/apply')
      }
      return
    }
    if (!access.hasApprovedApplication.value && to.path !== '/dashboard/apply') {
      return navigateTo('/dashboard/apply')
    }
    if (access.hasApprovedApplication.value && to.path === '/dashboard/apply') {
      return navigateTo('/dashboard')
    }
  }
})
