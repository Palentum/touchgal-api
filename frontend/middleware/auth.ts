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
    const isAdmin = auth.user.isAdmin
    const ok = await access.ensureApplications(auth.user.id)
    const hasApplicationAccess = access.hasApplicationGateAccess(isAdmin)
    if (!ok && !isAdmin) {
      if (to.path !== '/dashboard/apply') {
        return redirect('/dashboard/apply')
      }
      return
    }
    if (!hasApplicationAccess && to.path !== '/dashboard/apply') {
      return redirect('/dashboard/apply')
    }
    if (hasApplicationAccess && to.path === '/dashboard/apply') {
      return redirect('/dashboard')
    }
  }
})
