<template>
  <div class="grid gap-4 md:grid-cols-3 xl:grid-cols-6">
    <div v-for="card in cards" :key="card.label" class="rounded-3xl border border-white/10 bg-white/[0.06] p-5">
      <p class="text-sm text-slate-400">{{ card.label }}</p>
      <p class="mt-2 text-3xl font-black">{{ card.value }}</p>
    </div>
  </div>
</template>
<script setup lang="ts">
import type { StatsSummary } from '~/composables/useDashboard'
const props = defineProps<{ summary: StatsSummary }>()
const cards = computed(() => [
  { label: '总请求', value: props.summary.totalRequests },
  { label: '成功', value: props.summary.successRequests },
  { label: '错误', value: props.summary.errorRequests },
  { label: '平均耗时', value: `${props.summary.avgLatencyMs}ms` },
  { label: '来源', value: props.summary.uniqueOrigins },
  { label: 'IP', value: props.summary.uniqueIPs }
])
</script>
