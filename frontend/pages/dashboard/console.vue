<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">API Console</p>
      <h1 class="tg-display-md">API 调试台</h1>
      <p class="tg-lead">使用一次性输入的明文 token 验证 Public API 响应；页面不会保存 token。</p>
    </div>

    <DashboardApiConsole :tokens="tokens" />
  </section>
</template>

<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'

definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const dash = useDashboard()
const tokens = ref<TokenItem[]>([])

onMounted(async () => {
  const res = await dash.tokens()
  if (res.success) tokens.value = res.data
})
</script>
