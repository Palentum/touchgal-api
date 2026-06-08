<template>
  <section class="grid gap-6">
    <h1 class="text-4xl font-black">Token 管理</h1>
    <DashboardTokenCreateCard :applications="apps" @created="load" />
    <DashboardTokenTable :tokens="tokens" @revoke="revoke" />
  </section>
</template>
<script setup lang="ts">
import type { ApplicationItem, TokenItem } from '~/composables/useDashboard'
definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const dash = useDashboard()
const { apiFetch } = useApi()
const apps = ref<ApplicationItem[]>([])
const tokens = ref<TokenItem[]>([])
const load = async () => {
  const [appRes, tokenRes] = await Promise.all([dash.applications(), dash.tokens()])
  if (appRes.success) apps.value = appRes.data
  if (tokenRes.success) tokens.value = tokenRes.data
}
const revoke = async (id: string) => { await apiFetch(`/tokens/${id}/revoke`, { method: 'POST' }); await load() }
onMounted(load)
</script>
