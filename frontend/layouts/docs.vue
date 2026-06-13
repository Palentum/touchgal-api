<template>
  <div class="tg-dashboard-shell tg-docs-shell">
    <aside class="tg-dashboard-sidebar" :class="{ 'tg-sidebar-open': open }">
      <div class="tg-dashboard-sidebar-head">
        <NuxtLink to="/" class="tg-brand tg-brand-on-dark" @click="open = false">
          <img class="tg-logo" src="/logo.webp" width="32" height="32" alt="" aria-hidden="true">
          <span>TouchGal API</span>
        </NuxtLink>

        <button class="tg-icon-btn tg-mobile-toggle" type="button" :aria-expanded="open" aria-label="打开文档导航" @click="open = !open">
          {{ open ? '×' : '☰' }}
        </button>
      </div>

      <nav class="tg-sidebar-nav" aria-label="API 文档导航">
        <p class="tg-sidebar-section-label">API Reference</p>
        <NuxtLink to="/docs/api" class="tg-sidebar-link" @click="open = false">API 总览</NuxtLink>
        <NuxtLink
          v-for="doc in apiEndpointDocs"
          :key="doc.slug"
          :to="`/docs/api/${doc.slug}`"
          class="tg-sidebar-link"
          @click="open = false"
        >
          {{ doc.navLabel }}
        </NuxtLink>

      </nav>
    </aside>

    <div class="tg-dashboard-content">
      <header class="tg-dashboard-header">
        <div>
          <p class="tg-eyebrow" style="margin-bottom: 4px;">TouchGal Docs</p>
          <p style="margin: 0; color: var(--tg-muted); font-size: 14px;">独立脱敏 Galgame Metadata API</p>
        </div>
      </header>

      <main class="tg-dashboard-body">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiEndpointDocs } from '~/composables/apiDocs'

const route = useRoute()
const open = ref(false)

watch(() => route.fullPath, () => {
  open.value = false
})
</script>

<style scoped>
.tg-docs-shell {
  background: var(--tg-canvas);
}

.tg-sidebar-section-label {
  margin: 10px 12px 2px;
  color: var(--tg-on-dark-soft);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.12em;
  line-height: 1;
  text-transform: uppercase;
}

</style>
