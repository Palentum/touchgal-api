export default defineNuxtRouteMiddleware(async (to) => {
  const nuxtApp = useNuxtApp()
  const auth = useAuthStore()
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
  if (!auth.user.isAdmin) {
    return redirect('/dashboard')
  }
})
