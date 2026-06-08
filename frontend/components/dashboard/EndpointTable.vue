<template>
  <div class="rounded-3xl border border-white/10 bg-white/[0.06] p-6">
    <h3 class="text-xl font-black">Endpoint 调用统计</h3>
    <table class="mt-4 w-full text-left text-sm">
      <thead class="text-slate-400"><tr><th class="py-2">Route</th><th>请求</th><th>平均耗时</th><th>错误率</th></tr></thead>
      <tbody><tr v-for="item in data" :key="item.route" class="border-t border-white/10"><td class="py-3 font-mono">{{ item.route }}</td><td>{{ item.requests }}</td><td>{{ item.avgLatencyMs }}ms</td><td>{{ (item.errorRate * 100).toFixed(2) }}%</td></tr></tbody>
    </table>
  </div>
</template>
<script setup lang="ts">
import type { EndpointItem } from '~/composables/useDashboard'
defineProps<{ data: EndpointItem[] }>()
</script>
