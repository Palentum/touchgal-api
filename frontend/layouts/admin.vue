<template>
  <div class="tg-dashboard-shell">
    <aside class="tg-dashboard-sidebar">
      <NuxtLink to="/" class="tg-brand tg-brand-on-dark">
        <img class="tg-logo" src="/logo.webp" width="32" height="32" alt="" aria-hidden="true">
        <span>TouchGal API</span>
      </NuxtLink>

      <nav class="tg-sidebar-nav" aria-label="管理员后台导航">
        <NuxtLink v-for="item in items" :key="item.to" :to="item.to" class="tg-sidebar-link">
          {{ item.label }}
        </NuxtLink>
      </nav>

      <div style="margin-top: auto; padding-top: 32px;">
        <p class="tg-muted" style="margin: 0; font-size: 13px;">{{ auth.user?.email || '未登录' }}</p>
      </div>
    </aside>

    <div class="tg-dashboard-content">
      <header class="tg-dashboard-header">
        <div>
          <p class="tg-eyebrow" style="margin-bottom: 4px;">Admin Console</p>
          <p style="margin: 0; color: var(--tg-muted); font-size: 14px;">管理员后台</p>
        </div>
        <button class="tg-btn tg-btn-secondary" type="button" @click="logout">退出</button>
      </header>

      <main class="tg-dashboard-body">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
const auth = useAuthStore()
const items = [
  { to: '/admin', label: '管理员概览' },
  { to: '/admin/applications', label: '审核申请' },
  { to: '/admin/tokens', label: '全部 Token' },
  { to: '/admin/sync', label: '同步状态' }
]
const logout = async () => {
  await auth.logout()
  await navigateTo('/auth/login')
}
</script>
