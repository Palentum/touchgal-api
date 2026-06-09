<template>
  <section class="tg-dashboard-stack">
    <div>
      <p class="tg-eyebrow">API Tokens</p>
      <h1 class="tg-display-md">Token 管理</h1>
      <p class="tg-lead">创建、复制和管理 token。</p>
    </div>

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

const revoke = async (id: string) => {
  await apiFetch(`/tokens/${id}/revoke`, { method: 'POST' })
  await load()
}

onMounted(load)
</script>
