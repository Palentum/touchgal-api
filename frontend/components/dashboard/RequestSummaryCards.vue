<template>
  <div class="tg-grid-6">
    <div v-for="card in cards" :key="card.label" class="tg-stat-card">
      <p class="tg-stat-label">{{ card.label }}</p>
      <p class="tg-stat-value">{{ card.value }}</p>
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
