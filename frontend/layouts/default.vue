<template>
  <div class="tg-page">
    <header class="tg-nav-shell">
      <nav class="tg-container tg-nav" aria-label="主导航">
        <NuxtLink to="/" class="tg-brand" @click="open = false">
          <img class="tg-logo" src="/logo.webp" width="32" height="32" alt="" aria-hidden="true">
          <span>TouchGal API</span>
        </NuxtLink>

        <div class="tg-nav-links">
          <NuxtLink to="/docs/api">API 文档</NuxtLink>
          <a :href="techUrl" target="_blank" rel="noreferrer">主项目文档</a>
          <NuxtLink to="/dashboard/console">调试台</NuxtLink>
          <NuxtLink to="/dashboard/stats">请求统计</NuxtLink>
        </div>

        <div class="tg-nav-actions">
          <NuxtLink to="/auth/login" class="tg-link">登录</NuxtLink>
          <NuxtLink to="/auth/register" class="tg-btn tg-btn-primary">申请 API</NuxtLink>
        </div>

        <button class="tg-icon-btn tg-mobile-toggle" type="button" :aria-expanded="open" aria-label="打开导航" @click="open = !open">
          {{ open ? '×' : '☰' }}
        </button>
      </nav>
      <div v-if="open" class="tg-container tg-mobile-menu">
        <NuxtLink to="/docs/api" @click="open = false">API 文档</NuxtLink>
        <a :href="techUrl" target="_blank" rel="noreferrer" @click="open = false">主项目文档</a>
        <NuxtLink to="/auth/login" @click="open = false">登录控制台</NuxtLink>
        <NuxtLink to="/auth/register" @click="open = false">申请 API</NuxtLink>
      </div>
    </header>

    <main>
      <slot />
    </main>

    <footer class="tg-card-dark" style="border-radius: 0; padding: 64px 0;">
      <div class="tg-container tg-grid-4">
        <div>
          <NuxtLink to="/" class="tg-brand tg-brand-on-dark">
            <img class="tg-logo" src="/logo.webp" width="32" height="32" alt="" aria-hidden="true">
            <span>TouchGal API</span>
          </NuxtLink>
          <p class="tg-muted" style="margin-top: 16px; max-width: 320px;">独立、脱敏、可限流的 Galgame 元数据开发者 API。只暴露 clean DB 中的公开信息。</p>
        </div>
        <div>
          <p class="tg-title-sm" style="color: var(--tg-on-dark);">Product</p>
          <div class="tg-muted" style="display: grid; gap: 8px; margin-top: 12px;">
            <NuxtLink to="/docs/api">API 文档</NuxtLink>
            <NuxtLink to="/dashboard/console">API 调试台</NuxtLink>
            <NuxtLink to="/dashboard/tokens">Token 管理</NuxtLink>
          </div>
        </div>
        <div>
          <p class="tg-title-sm" style="color: var(--tg-on-dark);">Developer</p>
          <div class="tg-muted" style="display: grid; gap: 8px; margin-top: 12px;">
            <NuxtLink to="/auth/register">申请 API</NuxtLink>
            <NuxtLink to="/auth/login">登录</NuxtLink>
            <a :href="techUrl" target="_blank" rel="noreferrer">TouchGal 主项目</a>
          </div>
        </div>
        <div>
          <p class="tg-title-sm" style="color: var(--tg-on-dark);">Boundary</p>
          <p class="tg-muted" style="margin-top: 12px;">不返回主站用户、评论、资源下载链接、上传者 ID 或 source_* 内部字段。</p>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
const config = useRuntimeConfig()
const route = useRoute()
const open = ref(false)
const techUrl = computed(() => String(config.public.touchgalTechDocsUrl))
watch(() => route.fullPath, () => {
  open.value = false
})
</script>
