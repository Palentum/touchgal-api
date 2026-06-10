<template>
  <div class="tg-dashboard-shell">
    <aside class="tg-dashboard-sidebar">
      <NuxtLink to="/" class="tg-brand tg-brand-on-dark">
        <img class="tg-logo" src="/logo.webp" width="32" height="32" alt="" aria-hidden="true">
        <span>TouchGAL API</span>
      </NuxtLink>

      <nav v-if="!showPending" class="tg-sidebar-nav" aria-label="控制台导航">
        <NuxtLink v-for="item in items" :key="item.to" :to="item.to" class="tg-sidebar-link">
          {{ item.label }}
        </NuxtLink>
      </nav>

    </aside>

    <div class="tg-dashboard-content">
      <header class="tg-dashboard-header">
        <div>
          <p class="tg-eyebrow" style="margin-bottom: 4px;">Developer Console</p>
          <p style="margin: 0; color: var(--tg-muted); font-size: 14px;">开发者控制台</p>
        </div>
        <button class="tg-btn tg-btn-secondary" type="button" :disabled="showPending" @click="logout">退出</button>
      </header>

      <main class="tg-dashboard-body">
        <h1 v-if="showPending" class="tg-display-md">加载中...</h1>
        <slot v-else />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
const auth = useAuthStore()
const access = useApplicationAccess()

const applyItem = { to: '/dashboard/apply', label: '账号申请' }
const dashboardItems = [
  { to: '/dashboard', label: '概览' },
  { to: '/dashboard/tokens', label: 'Token 管理' },
  { to: '/dashboard/stats', label: '请求统计' },
  { to: '/dashboard/console', label: 'API 调试台' }
]
const showPending = computed(() => !auth.loaded || !auth.user || !access.checked.value)
const items = computed(() => {
  if (showPending.value) {
    return []
  }
  if (!access.hasApprovedApplication.value) {
    return [applyItem]
  }
  return dashboardItems
})
const logout = async () => {
  await auth.logout()
  await navigateTo('/auth/login')
}
</script>
