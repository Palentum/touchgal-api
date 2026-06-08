<template>
  <section class="grid gap-6">
    <h1 class="text-4xl font-black">全部 Token</h1>
    <DashboardTokenTable :tokens="tokens" @revoke="revoke" />
  </section>
</template>
<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'
definePageMeta({ layout: 'dashboard', middleware: 'admin' })
const { apiFetch } = useApi()
const tokens = ref<TokenItem[]>([])
const load = async () => { const res = await apiFetch<TokenItem[]>('/admin/tokens', { query: { page: 1, limit: 50 } }); if (res.success) tokens.value = res.data }
const revoke = async (id: string) => { await apiFetch(`/admin/tokens/${id}/revoke`, { method: 'POST' }); await load() }
onMounted(load)
</script>
