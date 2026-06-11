<template>
  <div class="tg-page">
    <header class="tg-nav-shell">
      <nav class="tg-container tg-nav" aria-label="文档导航">
        <NuxtLink to="/" class="tg-brand" @click="open = false">
          <img class="tg-logo" src="/logo.webp" width="32" height="32" alt="" aria-hidden="true">
          <span>TouchGal API</span>
        </NuxtLink>
        <div class="tg-nav-actions">
          <NuxtLink to="/docs/api" class="tg-link">API 文档</NuxtLink>
          <NuxtLink to="/dashboard" class="tg-btn tg-btn-secondary">控制台</NuxtLink>
        </div>
        <button class="tg-icon-btn tg-mobile-toggle" type="button" :aria-expanded="open" aria-label="打开文档导航" @click="open = !open">
          {{ open ? '×' : '☰' }}
        </button>
      </nav>
      <div v-if="open" class="tg-container tg-mobile-menu">
        <NuxtLink to="/docs/api" @click="open = false">API 文档</NuxtLink>
        <NuxtLink to="/docs/tech" @click="open = false">技术文档</NuxtLink>
        <NuxtLink to="/dashboard" @click="open = false">控制台</NuxtLink>
      </div>
    </header>
    <main class="tg-container-narrow tg-section-tight">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const open = ref(false)

watch(() => route.fullPath, () => {
  open.value = false
})
</script>
