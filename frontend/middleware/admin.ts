import { sanitizePostLoginRedirect } from '~/utils/redirect'

export default defineNuxtRouteMiddleware(async (to) => {
  const nuxtApp = useNuxtApp()
  const auth = useAuthStore()
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
  if (!auth.user.isAdmin) {
    return redirect('/dashboard')
  }
})
