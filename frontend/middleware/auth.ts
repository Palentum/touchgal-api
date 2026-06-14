import { sanitizePostLoginRedirect } from '~/utils/redirect'

export default defineNuxtRouteMiddleware(async (to) => {
  const nuxtApp = useNuxtApp()
  const auth = useAuthStore()
  const access = useApplicationAccess()
  const redirect = (path: string) => nuxtApp.runWithContext(() => navigateTo(path))

  if (!auth.loaded) {
    await auth.fetchMe()
  }
  if (auth.hasAuthFetchError && import.meta.client) {
    await auth.fetchMe({ refresh: true })
  }
  if (auth.hasAuthFetchError) {
    return abortNavigation(createError({ statusCode: 503, statusMessage: '认证服务暂时不可用' }))
  }
  if (auth.isAccountDisabled) {
    return redirect('/account-disabled')
  }
  if (!auth.user) {
    const target = sanitizePostLoginRedirect(to.fullPath)
    return redirect(`/auth/login?redirect=${encodeURIComponent(target)}`)
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
