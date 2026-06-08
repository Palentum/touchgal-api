<template>
  <div class="min-h-screen bg-slate-950 text-slate-100">
    <aside class="fixed inset-y-0 left-0 hidden w-64 border-r border-white/10 bg-slate-950/95 p-6 md:block">
      <NuxtLink to="/" class="text-xl font-black">TouchGal API</NuxtLink>
      <nav class="mt-8 grid gap-2 text-sm">
        <NuxtLink v-for="item in items" :key="item.to" :to="item.to" class="rounded-xl px-3 py-2 text-slate-300 hover:bg-white/10 hover:text-white">{{ item.label }}</NuxtLink>
      </nav>
    </aside>
    <div class="md:pl-64">
      <header class="flex items-center justify-between border-b border-white/10 px-6 py-4">
        <NuxtLink to="/" class="md:hidden">TouchGal API</NuxtLink>
        <div class="text-sm text-slate-400">控制台</div>
        <button class="rounded-full border border-white/10 px-4 py-2 text-sm" @click="logout">退出</button>
      </header>
      <main class="p-6">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const auth = useAuthStore()
const access = useApplicationAccess()
const applyItem = { to: '/dashboard/apply', label: '账号申请' }
const dashboardItems = [
  { to: '/dashboard', label: '概览' },
  { to: '/dashboard/tokens', label: 'Token 管理' },
  { to: '/dashboard/stats', label: '请求统计' },
  { to: '/dashboard/console', label: 'API 调试台' },
]
const items = computed(() => {
  if (!route.path.startsWith('/dashboard')) {
    return dashboardItems
  }
  return access.loaded.value && access.hasApprovedApplication.value ? dashboardItems : [applyItem]
})
onMounted(() => {
  if (route.path.startsWith('/dashboard') && auth.user) {
    void access.fetchApplications(auth.user.id, false, auth.user.isAdmin)
  }
})
const logout = async () => {
  await auth.logout()
  await navigateTo('/auth/login')
}
</script>
