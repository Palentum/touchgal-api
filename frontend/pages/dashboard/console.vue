<template>
  <DashboardApiConsole :tokens="tokens" />
</template>
<script setup lang="ts">
import type { TokenItem } from '~/composables/useDashboard'
definePageMeta({ layout: 'dashboard', middleware: 'auth' })
const dash = useDashboard()
const tokens = ref<TokenItem[]>([])
onMounted(async () => { const res = await dash.tokens(); if (res.success) tokens.value = res.data })
</script>
