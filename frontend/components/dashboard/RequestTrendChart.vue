<template>
  <ClientOnly>
    <div class="tg-chart-box">
      <VChart class="h-full min-h-[260px] w-full md:min-h-[320px]" :option="option" autoresize />
    </div>
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
const chartColors = {
  primary: '#cc785c',
  teal: '#5db8a6',
  error: '#c64545',
  muted: '#6c6a64',
  mutedSoft: '#8e8b82',
  hairline: 'rgba(230, 223, 216, 0.72)'
} as const
const chartPalette = [chartColors.primary, chartColors.teal, chartColors.error]

const option = computed<EChartsOption>(() => ({
  backgroundColor: 'transparent',
  color: chartPalette,
  tooltip: {
    trigger: 'axis',
    backgroundColor: '#181715',
    borderColor: '#252320',
    textStyle: { color: '#faf9f5' }
  },
  legend: { textStyle: { color: chartColors.muted } },
  grid: { left: 36, right: 18, bottom: 28, top: 40 },
  xAxis: { type: 'category', data: props.data.map((i) => i.date), axisLabel: { color: chartColors.mutedSoft } },
  yAxis: { type: 'value', axisLabel: { color: chartColors.mutedSoft }, splitLine: { lineStyle: { color: chartColors.hairline } } },
  series: [
    { name: '请求量', type: 'line', smooth: true, data: props.data.map((i) => i.totalRequests), lineStyle: { color: chartColors.primary } },
    { name: '成功', type: 'bar', stack: 'status', data: props.data.map((i) => i.successRequests), itemStyle: { color: chartColors.teal } },
    { name: '错误', type: 'bar', stack: 'status', data: props.data.map((i) => i.errorRequests), itemStyle: { color: chartColors.error } }
  ]
}))
</script>
