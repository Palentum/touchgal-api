<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">API Token</p>
      <h1 class="tg-display-md">全部 Token</h1>
      <p class="tg-lead">查看已签发的 API Token，并在需要时直接删除。</p>
    </div>
    <DashboardTokenTable :tokens="tokens" @delete="deleteToken" />
  </section>
</template>
<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'
definePageMeta({ layout: 'admin', middleware: 'admin' })
const { apiFetch, apiData } = useApi()
const { data: tokensResponse, refresh: refreshTokens } = await apiData<TokenItem[]>('admin:tokens', '/admin/tokens', { query: { page: 1, limit: 50 } })
const tokens = computed(() => tokensResponse.value?.success ? tokensResponse.value.data : [])
const load = async () => { await refreshTokens() }
const deleteToken = async (id: string) => { await apiFetch(`/admin/tokens/${id}`, { method: 'DELETE' }); await load() }
</script>
