<template>
  <div class="tg-card">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="tg-eyebrow">Endpoint Call Statistics</p>
        <h2 class="tg-title-lg">端点调用统计</h2>
      </div>
      <span class="tg-badge">{{ data.length }} 条路由</span>
    </div>

    <div class="tg-table-wrap mt-6" data-mobile-cards="true">
      <table class="tg-table">
        <thead>
          <tr>
            <th>Route</th>
            <th>请求</th>
            <th>平均耗时</th>
            <th>错误率</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in data" :key="item.route">
            <td class="font-mono" data-label="Route">{{ item.route }}</td>
            <td data-label="请求">{{ item.requests }}</td>
            <td data-label="平均耗时">{{ item.avgLatencyMs }}ms</td>
            <td data-label="错误率">{{ (item.errorRate * 100).toFixed(2) }}%</td>
          </tr>
        </tbody>
      </table>
      <p v-if="data.length === 0" class="tg-empty">暂无端点统计。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { EndpointItem } from '~/composables/useDashboard'

defineProps<{ data: EndpointItem[] }>()
</script>
