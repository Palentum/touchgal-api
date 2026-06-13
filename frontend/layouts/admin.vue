<template>
  <div class="tg-dashboard-shell">
    <aside class="tg-dashboard-sidebar" :class="{ 'tg-sidebar-open': navOpen }">
      <div class="tg-dashboard-sidebar-head">
        <NuxtLink to="/" class="tg-brand tg-brand-on-dark" @click="navOpen = false">
          <img class="tg-logo" src="/logo.webp" width="32" height="32" alt="" aria-hidden="true">
          <span>TouchGal API</span>
        </NuxtLink>

        <button class="tg-icon-btn tg-mobile-toggle" type="button" :aria-expanded="navOpen" aria-label="打开管理员导航" @click="navOpen = !navOpen">
          {{ navOpen ? '×' : '☰' }}
        </button>
      </div>

      <nav class="tg-sidebar-nav" aria-label="管理员后台导航">
        <NuxtLink v-for="item in items" :key="item.to" :to="item.to" class="tg-sidebar-link" @click="navOpen = false">
          {{ item.label }}
        </NuxtLink>
      </nav>

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
const route = useRoute()
const navOpen = ref(false)

const items = [
  { to: '/admin', label: '概览' },
  { to: '/admin/users', label: '用户管理' },
  { to: '/admin/applications', label: '申请审核' },
  { to: '/admin/tokens', label: '全部 Token' },
  { to: '/admin/sync', label: '同步状态' }
]
const logout = async () => {
  navOpen.value = false
  await auth.logout()
  await navigateTo('/auth/login')
}

watch(() => route.fullPath, () => {
  navOpen.value = false
})
</script>
