<template>
  <ClientOnly>
    <VChart class="h-80 w-full" :option="option" autoresize />
  </ClientOnly>
</template>
<script setup lang="ts">
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsOption } from 'echarts'
import VChart from 'vue-echarts'
import type { TrendItem } from '~/composables/useDashboard'
use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent])
const props = defineProps<{ data: TrendItem[] }>()
const option = computed<EChartsOption>(() => ({
  backgroundColor: 'transparent',
  tooltip: { trigger: 'axis' },
  legend: { textStyle: { color: '#cbd5e1' } },
  grid: { left: 36, right: 18, bottom: 28, top: 40 },
  xAxis: { type: 'category', data: props.data.map((i) => i.date), axisLabel: { color: '#94a3b8' } },
  yAxis: { type: 'value', axisLabel: { color: '#94a3b8' }, splitLine: { lineStyle: { color: 'rgba(148,163,184,.16)' } } },
  series: [
    { name: '请求量', type: 'line', smooth: true, data: props.data.map((i) => i.totalRequests), lineStyle: { color: '#34d399' } },
    { name: '成功', type: 'bar', stack: 'status', data: props.data.map((i) => i.successRequests), itemStyle: { color: '#38bdf8' } },
    { name: '错误', type: 'bar', stack: 'status', data: props.data.map((i) => i.errorRequests), itemStyle: { color: '#fb7185' } }
  ]
}))
</script>
