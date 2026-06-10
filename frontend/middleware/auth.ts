export default defineNuxtRouteMiddleware(async (to) => {
  const nuxtApp = useNuxtApp()
  const auth = useAuthStore()
  const access = useApplicationAccess()
  const redirect = (path: string) => nuxtApp.runWithContext(() => navigateTo(path))

  if (!auth.loaded) {
    await auth.fetchMe()
  }
  if (auth.isAccountDisabled) {
    return redirect('/account-disabled')
  }
  if (!auth.user) {
    return redirect(`/auth/login?redirect=${encodeURIComponent(to.fullPath)}`)
  }
  if (to.path.startsWith('/dashboard')) {
    const ok = await access.ensureApplications(auth.user.id)
    if (!ok) {
      if (to.path !== '/dashboard/apply') {
        return redirect('/dashboard/apply')
      }
      return
    }
    if (!access.hasApprovedApplication.value && to.path !== '/dashboard/apply') {
      return redirect('/dashboard/apply')
    }
    if (access.hasApprovedApplication.value && to.path === '/dashboard/apply') {
      return redirect('/dashboard')
    }
  }
})
