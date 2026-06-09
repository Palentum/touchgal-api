<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">API Token</p>
      <h1 class="tg-display-md">全部 Token</h1>
      <p class="tg-lead">查看已签发的 API Token，并在需要时执行吊销。</p>
    </div>
    <DashboardTokenTable :tokens="tokens" @revoke="revoke" />
  </section>
</template>
<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'
definePageMeta({ layout: 'admin', middleware: 'admin' })
const { apiFetch } = useApi()
const tokens = ref<TokenItem[]>([])
const load = async () => { const res = await apiFetch<TokenItem[]>('/admin/tokens', { query: { page: 1, limit: 50 } }); if (res.success) tokens.value = res.data }
const revoke = async (id: string) => { await apiFetch(`/admin/tokens/${id}/revoke`, { method: 'POST' }); await load() }
onMounted(load)
</script>
